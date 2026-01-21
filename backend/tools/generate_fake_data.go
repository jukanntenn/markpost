package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
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

var bodyTexts = []string{
	"在现代软件开发中,合理的技术选型对项目的成功至关重要。本文将从多个角度分析各种技术方案的优缺点,帮助开发者做出更明智的决策。通过实际案例的对比,我们可以看到不同架构模式在不同场景下的表现。",
	"并发编程是Go语言的核心特性之一。通过Goroutine和Channel,我们可以轻松实现高效的并发处理,充分利用多核CPU的计算能力。本文将详细介绍Go并发模型的原理,并通过实例演示如何避免常见的并发陷阱。",
	"数据库性能优化是后端开发中的重要话题。合理的索引设计、查询优化、读写分离等策略都能显著提升系统的响应速度和吞吐量。我们将深入探讨SQL优化的技巧,以及如何在保证数据一致性的前提下提升性能。",
	"微服务架构是现代分布式系统的重要设计模式。通过将单体应用拆分为多个小型服务,我们可以实现更好的可扩展性和可维护性。然而,微服务也带来了分布式事务、服务发现、链路追踪等新的挑战。",
	"容器化技术改变了应用的部署方式。Docker提供了轻量级的虚拟化解决方案,而Kubernetes则提供了强大的容器编排能力。本文将介绍如何使用这些工具构建现代化的云原生应用。",
	"RESTful API设计是Web开发的基础。良好的API设计应该遵循资源导向的原则,使用合适的HTTP方法和状态码。我们将讨论如何设计清晰、一致、易于使用的API接口。",
	"缓存是提升系统性能的有效手段。通过在内存中存储热点数据,可以大幅减轻数据库的压力。Redis作为流行的缓存解决方案,提供了丰富的数据结构和强大的功能。",
	"消息队列是实现系统解耦和异步处理的重要工具。通过引入消息队列,我们可以将同步调用转换为异步处理,提升系统的吞吐量和容错能力。RabbitMQ和Kafka是两种常用的消息中间件。",
	"测试是保证软件质量的关键环节。单元测试、集成测试、端到端测试构成了完整的测试体系。本文将介绍Go语言中的测试最佳实践,包括表驱动测试、测试替身等技巧。",
	"算法与数据结构是计算机科学的基石。掌握常用的算法和数据结构,能够帮助我们编写更高效的代码。我们将通过实际问题,演示如何选择和实现合适的算法方案。",
	"网络安全是Web应用不可忽视的方面。HTTPS、CSRF防护、XSS防护、SQL注入防护等都是构建安全应用必须要考虑的问题。本文将介绍常见的安全漏洞及其防御措施。",
	"前端技术发展迅速,React、Vue、Angular等框架各具特色。选择合适的前端框架,并遵循组件化、状态管理等最佳实践,可以大幅提升开发效率和代码质量。",
	"负载均衡是高可用系统的核心组件。通过将流量分发到多个服务器,可以实现系统的水平扩展。我们将介绍常用的负载均衡算法,以及Nginx等负载均衡器的配置方法。",
	"日志和监控是运维系统的重要组成部分。通过收集和分析日志数据,我们可以及时发现和定位问题。Prometheus和Grafana是流行的监控解决方案,可以提供全面的系统可观测性。",
	"代码审查是提升代码质量的有效手段。通过团队成员之间的相互审查,可以发现潜在的问题,分享知识经验,建立统一的编码规范。本文将介绍代码审查的流程和注意事项。",
}

func main() {
	outputFile := "fake.json"
	if len(os.Args) > 1 {
		outputFile = os.Args[1]
	}

	posts := make([]FakePost, 100)
	now := time.Now()
	ninetyDaysAgo := now.AddDate(0, 0, -90)

	for i := 0; i < 100; i++ {
		qid, _ := gonanoid.New()

		randomHours, _ := rand.Int(rand.Reader, big.NewInt(90*24))
		createdAt := ninetyDaysAgo.Add(time.Duration(randomHours.Int64()) * time.Hour)

		randomMins, _ := rand.Int(rand.Reader, big.NewInt(1440))
		updatedAt := createdAt.Add(time.Duration(randomMins.Int64()) * time.Minute)

		title := generateTitle(10, 50)
		body := generateBody(100, 500)

		posts[i] = FakePost{
			QID:       qid,
			Title:     title,
			Body:      body,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		}
	}

	data, err := json.Marshal(posts)
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(outputFile, data, 0644); err != nil {
		fmt.Printf("Error writing file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully generated %d fake posts to %s\n", len(posts), outputFile)
}

func generateTitle(minLen, maxLen int) string {
	title := ""
	for len(title) < minLen || len(title) > maxLen {
		title = ""
		wordCount, _ := rand.Int(rand.Reader, big.NewInt(5))
		wordCountInt := int(wordCount.Int64()) + 3

		for i := 0; i < wordCountInt; i++ {
			idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(titleWords))))
			title += titleWords[idx.Int64()]
		}
	}
	return truncateByBytes(title, maxLen)
}

func generateBody(minLen, maxLen int) string {
	body := ""
	for len(body) < minLen {
		idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(bodyTexts))))
		body += bodyTexts[idx.Int64()] + "\n\n"
	}
	return truncateByBytes(body, maxLen)
}

func truncateByBytes(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	return string([]byte(s)[:maxBytes])
}
