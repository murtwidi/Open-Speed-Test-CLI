package main

import (
	"crypto/rand"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ─── ANSI colours ────────────────────────────────────────────────────────────

const (
	reset   = "\033[0m"
	bold    = "\033[1m"
	dim     = "\033[2m"
	red     = "\033[31m"
	green   = "\033[32m"
	yellow  = "\033[33m"
	blue    = "\033[34m"
	magenta = "\033[35m"
	cyan    = "\033[36m"
	white   = "\033[37m"
)

// ─── Config ───────────────────────────────────────────────────────────────────

const (
	defaultHost     = "localhost"
	defaultPort     = 3000
	downloadPath    = "/downloading"
	uploadPath      = "/upload"
	defaultThreads  = 6
	defaultDuration = 10 * time.Second
	pingSamples     = 20
	overheadComp    = 1.04  // +4% overhead compensation, same as browser
	chunkSize       = 65536 // 64 KB read buffer
	uploadPayloadMB = 32    // random payload size per upload thread (MB)
)

// ─── HTTP client ──────────────────────────────────────────────────────────────
// Shared client with no body-read timeout; TLS verification disabled for
// self-hosted servers that may use self-signed certificates.

func newClient(dur time.Duration) *http.Client {
	tr := &http.Transport{
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		MaxIdleConnsPerHost: defaultThreads * 2,
		DisableCompression:  true,
	}
	return &http.Client{Transport: tr, Timeout: dur + 5*time.Second}
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// buildURL normalises the user-supplied host + port into a full base URL.
func buildURL(host string, port int) string {
	if strings.HasPrefix(host, "http://") || strings.HasPrefix(host, "https://") {
		return strings.TrimRight(host, "/")
	}
	return fmt.Sprintf("http://%s:%d", host, port)
}

// fmtSpeed formats Mbps into a human-readable speed string.
func fmtSpeed(mbps float64) string {
	switch {
	case mbps >= 1000:
		return fmt.Sprintf("%.2f Gbps", mbps/1000)
	case mbps >= 1:
		return fmt.Sprintf("%.2f Mbps", mbps)
	default:
		return fmt.Sprintf("%.2f Kbps", mbps*1000)
	}
}

func fmtMS(ms float64) string { return fmt.Sprintf("%.2f ms", ms) }

// calcMbps converts raw bytes and elapsed seconds to Mbps, applying the
// overhead compensation factor to match browser-based results.
func calcMbps(bytes int64, elapsed float64) float64 {
	return (float64(bytes) * 8 / elapsed / 1e6) * overheadComp
}

// ─── Spinner ──────────────────────────────────────────────────────────────────

type spinner struct {
	done  chan struct{}
	label string
}

func newSpinner(label string) *spinner {
	s := &spinner{done: make(chan struct{}), label: label}
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	go func() {
		i := 0
		for {
			select {
			case <-s.done:
				return
			default:
				fmt.Printf("\r  %s%s%s %s%s%s     ",
					cyan, frames[i%len(frames)], reset,
					bold, s.label, reset)
				time.Sleep(80 * time.Millisecond)
				i++
			}
		}
	}()
	return s
}

func (s *spinner) stop() {
	close(s.done)
	fmt.Print("\r\033[K") // clear the spinner line
}

// ─── Live progress bar ────────────────────────────────────────────────────────
// Runs in a goroutine alongside the test; reads the live byte counter and
// displays a progress bar with the current speed estimate.

func liveProgress(label string, dur time.Duration, bytesRef *int64) {
	const barWidth = 28
	start := time.Now()
	for {
		elapsed := time.Since(start)
		if elapsed >= dur {
			fmt.Print("\r\033[K")
			return
		}
		frac := elapsed.Seconds() / dur.Seconds()
		filled := int(frac * float64(barWidth))
		bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

		current := atomic.LoadInt64(bytesRef)
		mbps := 0.0
		if elapsed.Seconds() > 0 {
			mbps = calcMbps(current, elapsed.Seconds())
		}
		pct := int(frac * 100)

		fmt.Printf("\r  %s%s%s  [%s%s%s]  %3d%%  %s%-14s%s",
			bold+cyan, label, reset,
			green, bar, reset,
			pct,
			yellow, fmtSpeed(mbps), reset,
		)
		time.Sleep(150 * time.Millisecond)
	}
}

// ─── Ping test ────────────────────────────────────────────────────────────────

type pingResult struct {
	Min, Avg, Max, Jitter float64
	Loss                  float64
}

// runPing measures latency by sending HEAD requests to the download endpoint
// and recording round-trip times.
func runPing(baseURL string) pingResult {
	url := baseURL + downloadPath
	client := newClient(5 * time.Second)
	var rtts []float64

	for i := 0; i < pingSamples; i++ {
		req, _ := http.NewRequest("HEAD", url, nil)
		t0 := time.Now()
		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
			rtts = append(rtts, float64(time.Since(t0).Microseconds())/1000.0)
		}
	}

	if len(rtts) == 0 {
		return pingResult{Loss: 100}
	}

	sort.Float64s(rtts)
	sum := 0.0
	for _, v := range rtts {
		sum += v
	}
	avg := sum / float64(len(rtts))

	// Jitter = standard deviation of RTT samples
	variance := 0.0
	for _, v := range rtts {
		d := v - avg
		variance += d * d
	}
	jitter := math.Sqrt(variance / float64(len(rtts)))
	loss := float64(pingSamples-len(rtts)) / pingSamples * 100

	return pingResult{
		Min:    rtts[0],
		Avg:    avg,
		Max:    rtts[len(rtts)-1],
		Jitter: jitter,
		Loss:   loss,
	}
}

// ─── Download test ────────────────────────────────────────────────────────────

type speedResult struct {
	Mbps    float64
	Bytes   int64
	Elapsed float64
	Threads int
}

// runDownload opens `threads` parallel GET streams to the download endpoint
// and counts every byte received during `dur`.
func runDownload(baseURL string, dur time.Duration, threads int) speedResult {
	url := baseURL + downloadPath
	client := newClient(dur)

	var totalBytes int64
	var wg sync.WaitGroup
	deadline := time.Now().Add(dur)

	for i := 0; i < threads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			buf := make([]byte, chunkSize)
			for time.Now().Before(deadline) {
				resp, err := client.Get(url)
				if err != nil {
					continue
				}
				for time.Now().Before(deadline) {
					n, err := resp.Body.Read(buf)
					if n > 0 {
						atomic.AddInt64(&totalBytes, int64(n))
					}
					if err != nil {
						break
					}
				}
				resp.Body.Close()
			}
		}()
	}

	t0 := time.Now()
	go liveProgress("Download", dur, &totalBytes)
	wg.Wait()
	elapsed := time.Since(t0).Seconds()

	return speedResult{
		Mbps:    calcMbps(atomic.LoadInt64(&totalBytes), elapsed),
		Bytes:   atomic.LoadInt64(&totalBytes),
		Elapsed: elapsed,
		Threads: threads,
	}
}

// ─── Upload test ──────────────────────────────────────────────────────────────

// infiniteReader streams a pre-filled random buffer in a circular fashion
// until the deadline is reached, avoiding repeated allocations.
type infiniteReader struct {
	buf      []byte
	pos      int
	deadline time.Time
}

func newInfiniteReader(deadline time.Time) *infiniteReader {
	buf := make([]byte, uploadPayloadMB*1024*1024)
	rand.Read(buf)
	return &infiniteReader{buf: buf, deadline: deadline}
}

func (r *infiniteReader) Read(p []byte) (int, error) {
	if time.Now().After(r.deadline) {
		return 0, io.EOF
	}
	n := copy(p, r.buf[r.pos:])
	r.pos = (r.pos + n) % len(r.buf)
	return n, nil
}

// runUpload opens `threads` parallel POST streams to the upload endpoint and
// counts every byte sent during `dur`.
func runUpload(baseURL string, dur time.Duration, threads int) speedResult {
	url := baseURL + uploadPath
	client := newClient(dur)

	var totalBytes int64
	var wg sync.WaitGroup
	deadline := time.Now().Add(dur)

	for i := 0; i < threads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for time.Now().Before(deadline) {
				reader := newInfiniteReader(deadline)
				cr := &countingReader{r: reader, total: &totalBytes}
				req, err := http.NewRequest("POST", url, cr)
				if err != nil {
					continue
				}
				req.Header.Set("Content-Type", "application/octet-stream")
				req.ContentLength = -1 // use chunked transfer encoding
				resp, err := client.Do(req)
				if err == nil {
					resp.Body.Close()
				}
			}
		}()
	}

	t0 := time.Now()
	go liveProgress("Upload  ", dur, &totalBytes)
	wg.Wait()
	elapsed := time.Since(t0).Seconds()

	return speedResult{
		Mbps:    calcMbps(atomic.LoadInt64(&totalBytes), elapsed),
		Bytes:   atomic.LoadInt64(&totalBytes),
		Elapsed: elapsed,
		Threads: threads,
	}
}

// countingReader wraps an io.Reader and atomically accumulates the total
// number of bytes read into *total.
type countingReader struct {
	r     io.Reader
	total *int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	if n > 0 {
		atomic.AddInt64(c.total, int64(n))
	}
	return n, err
}

// ─── Server reachability check ────────────────────────────────────────────────

func checkServer(baseURL string) bool {
	client := newClient(5 * time.Second)
	req, _ := http.NewRequest("HEAD", baseURL+downloadPath, nil)
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode < 500
}

// ─── Output helpers ───────────────────────────────────────────────────────────

func printBanner(baseURL string) {
	line := strings.Repeat("─", 60)
	fmt.Printf("\n%s%s%s\n", bold+blue, line, reset)
	fmt.Printf("%s  ospeedtest  —  OpenSpeedTest CLI  →  %s%s\n", bold+white, baseURL, reset)
	fmt.Printf("%s%s%s\n\n", dim+blue, line, reset)
}

func printRow(icon, label, value, detail string) {
	fmt.Printf("  %s  %-12s%s%-18s%s  %s%s%s\n",
		icon, label,
		bold+green, value, reset,
		dim, detail, reset,
	)
}

func printResults(ping *pingResult, dl, ul *speedResult) {
	line := strings.Repeat("─", 60)

	fmt.Printf("\n%s%s%s\n", bold+blue, line, reset)
	fmt.Printf("%s  %-14s%-20s%s\n", bold+white, "  Test", "Result          Detail", reset)
	fmt.Printf("%s%s%s\n", dim+blue, line, reset)

	if ping != nil {
		if ping.Loss < 100 {
			printRow("📡", "Ping (avg)",
				fmtMS(ping.Avg),
				fmt.Sprintf("min %s / max %s", fmtMS(ping.Min), fmtMS(ping.Max)),
			)
			printRow("〰 ", "Jitter",
				fmtMS(ping.Jitter),
				fmt.Sprintf("packet loss: %.1f%%", ping.Loss),
			)
		} else {
			printRow("📡", "Ping", red+"FAILED"+reset, "server did not respond")
		}
	}

	if dl != nil {
		printRow("⬇ ", "Download",
			bold+green+fmtSpeed(dl.Mbps)+reset,
			fmt.Sprintf("%.1f MB in %.1fs via %d threads", float64(dl.Bytes)/1e6, dl.Elapsed, dl.Threads),
		)
	}

	if ul != nil {
		printRow("⬆ ", "Upload",
			bold+cyan+fmtSpeed(ul.Mbps)+reset,
			fmt.Sprintf("%.1f MB in %.1fs via %d threads", float64(ul.Bytes)/1e6, ul.Elapsed, ul.Threads),
		)
	}

	fmt.Printf("%s%s%s\n", dim+blue, line, reset)
	fmt.Printf("  %s+4%% overhead compensation applied (matches browser behaviour)%s\n\n", dim, reset)
}

// ─── Main ─────────────────────────────────────────────────────────────────────

func main() {
	host     := flag.String("host", defaultHost, "Server IP, hostname, or full URL")
	port     := flag.Int("port", defaultPort, "Server port")
	dur      := flag.Duration("duration", defaultDuration, "Duration of each test (e.g. 10s, 30s)")
	threads  := flag.Int("threads", defaultThreads, "Number of parallel connections")
	pingOnly := flag.Bool("ping", false, "Run ping test only")
	dlOnly   := flag.Bool("download", false, "Run download test only")
	ulOnly   := flag.Bool("upload", false, "Run upload test only")
	noColor  := flag.Bool("no-color", false, "Disable ANSI colour output")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `
%sospeedtest%s — OpenSpeedTest CLI  (single binary, no runtime required)

%sUsage:%s
  ospeedtest [flags]

%sFlags:%s
`, bold, reset, bold, reset, bold, reset)
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
%sExamples:%s
  ospeedtest -host 192.168.1.5
  ospeedtest -host 192.168.1.5 -port 8080
  ospeedtest -host http://192.168.1.5:3000
  ospeedtest -host 192.168.1.5 -download
  ospeedtest -host 192.168.1.5 -upload
  ospeedtest -host 192.168.1.5 -ping
  ospeedtest -host 192.168.1.5 -duration 30s -threads 8
  ospeedtest -host 192.168.1.5 -no-color

`, bold, reset)
	}

	flag.Parse()

	// Accept bare positional arguments: ospeedtest 192.168.1.5 8080
	if flag.NArg() > 0 {
		*host = flag.Arg(0)
	}
	if flag.NArg() > 1 {
		fmt.Sscanf(flag.Arg(1), "%d", port)
	}

	if *noColor {
		os.Setenv("NO_COLOR", "1")
	}

	baseURL := buildURL(*host, *port)

	// Determine which tests to run
	doPing := !*dlOnly && !*ulOnly
	doDL   := !*pingOnly && !*ulOnly
	doUL   := !*pingOnly && !*dlOnly

	printBanner(baseURL)

	// ── Check server reachability ──────────────────────────────────────────
	sp := newSpinner("Checking server...")
	ok := checkServer(baseURL)
	sp.stop()
	if !ok {
		fmt.Printf("  %s✗ Could not reach server at %s%s\n\n", red+bold, baseURL, reset)
		os.Exit(1)
	}
	fmt.Printf("  %s✓ Server reachable%s\n\n", green+bold, reset)

	var (
		pingRes *pingResult
		dlRes   *speedResult
		ulRes   *speedResult
	)

	// ── Ping ──────────────────────────────────────────────────────────────
	if doPing {
		fmt.Printf("  %s[1/3] Ping Test%s  (%d samples)\n", bold+magenta, reset, pingSamples)
		sp = newSpinner(fmt.Sprintf("Sending %d ping requests...", pingSamples))
		r := runPing(baseURL)
		sp.stop()
		pingRes = &r
		if r.Loss < 100 {
			fmt.Printf("  %s✓%s  avg %s%s%s  jitter %s\n\n",
				green, reset, bold+yellow, fmtMS(r.Avg), reset, fmtMS(r.Jitter))
		} else {
			fmt.Printf("  %s✗ Ping failed — no response from server%s\n\n", red, reset)
		}
	}

	// ── Download ──────────────────────────────────────────────────────────
	if doDL {
		fmt.Printf("  %s[2/3] Download Test%s  (%v / %d threads)\n",
			bold+magenta, reset, *dur, *threads)
		r := runDownload(baseURL, *dur, *threads)
		dlRes = &r
		fmt.Printf("  %s✓%s  %s%s%s\n\n",
			green, reset, bold+green, fmtSpeed(r.Mbps), reset)
	}

	// ── Upload ────────────────────────────────────────────────────────────
	if doUL {
		fmt.Printf("  %s[3/3] Upload Test%s  (%v / %d threads)\n",
			bold+magenta, reset, *dur, *threads)
		r := runUpload(baseURL, *dur, *threads)
		ulRes = &r
		fmt.Printf("  %s✓%s  %s%s%s\n\n",
			green, reset, bold+cyan, fmtSpeed(r.Mbps), reset)
	}

	printResults(pingRes, dlRes, ulRes)
}
