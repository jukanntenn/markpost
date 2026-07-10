// Command generate_fake_data generates fake post data as JSON for seeding a
// database prior to load testing. The output is consumed by the
// `import-fake-posts` server subcommand.
//
// Body size, post count, and the random seed are flag-configurable so a run is
// reproducible: the same seed yields the same QIDs and bodies, which keeps
// load-test target files stable across regenerations.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"
)

type FakePost struct {
	QID       string    `json:"qid"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

var titleWords = []string{
	"关于", "深入", "探讨", "分析", "研究", "实现", "设计", "开发",
	"Go语言", "编程", "并发", "数据库", "性能优化", "最佳实践",
	"微服务", "架构", "算法", "数据结构", "网络", "安全", "测试",
	"容器", "云原生", "分布式", "缓存", "消息队列", "负载均衡",
	"前端", "后端", "全栈", "DevOps", "CI/CD", "API", "REST",
	"GraphQL", "WebSocket", "gRPC", "protobuf", "JSON", "XML",
	"Linux", "Docker", "Kubernetes", "nginx", "redis", "mysql",
}

// markdownBlocks are reusable markdown fragments exercising the renderer
// (headings, lists, fenced code, tables, links, emphasis). generateBody tiles
// them until the target byte budget is reached so the rendered output stresses
// goldmark + bluemonday + minify realistically rather than being a flat wall.
var markdownBlocks = []string{
	"## 深入分析\n\n现代软件开发中，合理的技术选型对项目的成功至关重要。" +
		"本文将从多个角度分析各种技术方案的优缺点，帮助开发者做出更明智的决策。\n\n" +
		"- 第一要点涉及并发模型的取舍\n" +
		"- 第二要点关注数据一致性的边界\n" +
		"- 第三要点讨论可观测性与运维成本\n\n" +
		"| 维度 | 方案A | 方案B |\n|------|-------|-------|\n| 延迟 | 低 | 中 |\n| 成本 | 高 | 低 |\n\n",
	"### 代码示例\n\n" +
		"```go\nfunc process(ctx context.Context, in Input) (Output, error) {\n" +
		"    select {\n    case <-ctx.Done():\n        return Output{}, ctx.Err()\n    default:\n    }\n" +
		"    return transform(in), nil\n}\n```\n\n" +
		"该实现遵循 [Go 并发最佳实践](https://go.dev/doc/) 并使用 `context` 控制生命周期。\n\n",
	"#### 设计权衡\n\n" +
		"在 **吞吐量** 与 **延迟** 之间存在固有的权衡。" +
		"通过 ~~过度优化~~ 提前优化往往会引入不必要的复杂度。\n\n" +
		">  simplicity is the ultimate sophistication.\n\n",
	"##### 扩展阅读\n\n" +
		"1. 分布式系统的 CAP 定理及其现代诠释\n" +
		"2. 缓存失效策略与一致性模型\n" +
		"3. 消息队列的恰好一次语义\n\n" +
		"参考 [这篇文章](https://example.com/deep-dive) 获取更多细节。\n\n",
}

func main() {
	count := flag.Int("count", 100, "number of fake posts to generate")
	bodyBytes := flag.Int("body-bytes", 32768, "target body size in bytes per post (approximate)")
	output := flag.String("output", "fake.json", "output JSON file path")
	seed := flag.Int64("seed", 0, "random seed (0 = non-deterministic; fixed value = reproducible output)")
	flag.Parse()

	if *count <= 0 {
		fmt.Fprintln(os.Stderr, "count must be positive")
		os.Exit(1)
	}
	if *bodyBytes <= 0 {
		fmt.Fprintln(os.Stderr, "body-bytes must be positive")
		os.Exit(1)
	}

	// A fixed seed makes QIDs and body content reproducible across runs, which
	// keeps load-test target files stable. math/rand suffices here: this is
	// synthetic test data, not a security primitive.
	rng := rand.New(rand.NewSource(*seed))

	posts := make([]FakePost, *count)
	now := time.Now()
	windowStart := now.AddDate(0, 0, -90)

	for i := 0; i < *count; i++ {
		qid := generateQID(rng)

		ageHours := rng.Intn(90 * 24)
		createdAt := windowStart.Add(time.Duration(ageHours) * time.Hour)
		updatedAt := createdAt.Add(time.Duration(rng.Intn(1440)) * time.Minute)

		posts[i] = FakePost{
			QID:       qid,
			Title:     generateTitle(rng, 10, 50),
			Body:      generateBody(rng, *bodyBytes),
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		}
	}

	data, err := json.Marshal(posts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(*output, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "write %s: %v\n", *output, err)
		os.Exit(1)
	}

	fmt.Printf("Generated %d fake posts (%d-byte bodies) to %s\n", len(posts), *bodyBytes, *output)
}

// qidAlphabet mirrors nanoid's default URL-safe alphabet. generateQID draws
// from the seeded rng so QIDs are reproducible under a fixed seed (gonanoid
// itself sources crypto/rand and cannot be seeded).
const qidAlphabet = "_-0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func generateQID(rng *rand.Rand) string {
	b := make([]byte, 21)
	for i := range b {
		b[i] = qidAlphabet[rng.Intn(len(qidAlphabet))]
	}
	return string(b)
}

func generateTitle(rng *rand.Rand, minLen, maxLen int) string {
	var b strings.Builder
	for b.Len() < minLen {
		b.WriteString(titleWords[rng.Intn(len(titleWords))])
	}
	s := b.String()
	if len(s) > maxLen {
		// Truncate on a rune boundary to avoid splitting a multibyte char.
		r := []rune(s)
		maxRunes := maxLen
		if maxRunes > len(r) {
			maxRunes = len(r)
		}
		// Byte cap may still land mid-rune; trim back to the last full rune.
		for len(string(r[:maxRunes])) > maxLen && maxRunes > 0 {
			maxRunes--
		}
		return string(r[:maxRunes])
	}
	return s
}

// generateBody tiles markdownBlocks until the accumulated length meets or
// exceeds the target byte budget n, returning the full blocks without
// truncation. Truncating mid-block (and mid-multibyte rune) would produce
// invalid UTF-8 and unbalanced markdown fences; stopping at the block boundary
// keeps the body well-formed at a size approximately >= n.
func generateBody(rng *rand.Rand, n int) string {
	var b strings.Builder
	for b.Len() < n {
		b.WriteString(markdownBlocks[rng.Intn(len(markdownBlocks))])
	}
	return b.String()
}
