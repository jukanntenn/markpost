// Command generate_write_targets emits vegeta JSON-format targets for the
// write path (POST /:post_key) to stdout, one per line. The post body of each
// target is drawn from a normal distribution around the spec's 32 KiB average,
// so the write load reflects realistic post-size spread rather than a single
// fixed payload.
//
// Post-keys are read from a file (one per line, as produced by seed-users) and
// round-robined across targets so the L2 per-user rate limit is distributed.
package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"
)

// markdownBlocks tiles into a body of a requested byte length, keeping the
// markdown structure (headings, lists, tables, code) that goldmark, bluemonday
// and the delivery keyword filter must all process.
var markdownBlocks = []string{
	"## 深入分析\n\n技术选型对项目成功至关重要。" +
		"本文分析各种方案的优缺点。\n\n" +
		"- 第一要点\n- 第二要点\n- 第三要点\n\n" +
		"| 维度 | A | B |\n|------|---|---|\n| 延迟 | 低 | 中 |\n\n",
	"### 代码\n\n```go\nfunc f(x int) int { return x * 2 }\n```\n\n" +
		"参考 [文档](https://example.com) 和 `内联代码`。\n\n",
	"#### 权衡\n\n**吞吐量** 与 **延迟** 存在权衡。" +
		"~~过度优化~~ 提前优化引入复杂度。\n\n> 简单是终极的复杂。\n\n",
	"##### 列表\n\n1. CAP 定理\n2. 缓存失效\n3. 恰好一次语义\n\n",
}

type vegetaTarget struct {
	Method string              `json:"method"`
	URL    string              `json:"url"`
	Body   string              `json:"body"`
	Header map[string][]string `json:"header"`
}

type postPayload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

func main() {
	count := flag.Int("count", 1000, "number of targets to emit")
	keysFile := flag.String("keys-file", "", "file with one post_key per line (required)")
	host := flag.String("host", "localhost", "target host")
	port := flag.Int("port", 7330, "target port")
	mean := flag.Int("mean-bytes", 32768, "mean body size in bytes (normal distribution)")
	stddev := flag.Int("stddev-bytes", 8192, "body size standard deviation in bytes")
	seed := flag.Int64("seed", 1, "RNG seed (reproducible output)")
	flag.Parse()

	if *keysFile == "" {
		fmt.Fprintln(os.Stderr, "--keys-file is required")
		os.Exit(1)
	}

	keys, err := readLines(*keysFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read keys: %v\n", err)
		os.Exit(1)
	}
	if len(keys) == 0 {
		fmt.Fprintln(os.Stderr, "keys file is empty")
		os.Exit(1)
	}

	rng := rand.New(rand.NewSource(*seed))
	baseURL := fmt.Sprintf("http://%s:%d", *host, *port)
	enc := base64.StdEncoding
	encoder := json.NewEncoder(os.Stdout)

	for i := 0; i < *count; i++ {
		size := normalBodySize(rng, *mean, *stddev)
		payload := postPayload{
			Title: fmt.Sprintf("Load test post %d", i),
			Body:  generateBody(rng, size),
		}
		raw, err := json.Marshal(payload)
		if err != nil {
			fmt.Fprintf(os.Stderr, "marshal payload: %v\n", err)
			os.Exit(1)
		}

		key := keys[i%len(keys)]
		target := vegetaTarget{
			Method: "POST",
			URL:    fmt.Sprintf("%s/%s", baseURL, key),
			Body:   enc.EncodeToString(raw),
			Header: map[string][]string{"Content-Type": {"application/json"}},
		}
		if err := encoder.Encode(&target); err != nil {
			fmt.Fprintf(os.Stderr, "encode target: %v\n", err)
			os.Exit(1)
		}
	}
}

// normalBodySize draws from a normal distribution, clamped to [1 KiB, 32 KiB].
// The 32 KiB ceiling matches the post body validator; anything larger would be
// rejected with 400 before reaching the service layer.
func normalBodySize(rng *rand.Rand, mean, stddev int) int {
	const minBytes = 1024
	const maxBytes = 32768
	for {
		v := int(rng.NormFloat64()*float64(stddev) + float64(mean))
		if v >= minBytes && v <= maxBytes {
			return v
		}
	}
}

func generateBody(rng *rand.Rand, n int) string {
	var b strings.Builder
	for b.Len() < n {
		b.WriteString(markdownBlocks[rng.Intn(len(markdownBlocks))])
	}
	s := b.String()
	if len(s) > n {
		// Trim back to a rune boundary so the JSON body stays valid UTF-8.
		r := []rune(s)
		for len(string(r)) > n && len(r) > 0 {
			r = r[:len(r)-1]
		}
		return string(r)
	}
	return s
}

func readLines(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var lines []string
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines, nil
}
