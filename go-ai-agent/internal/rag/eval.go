package rag

import (
	"bufio"
	"encoding/json"
	"fmt"
	"go-ai-agent/internal/utils"
	"log"
	"os"

	"golang.org/x/exp/slices"
)

// EvalCheckConfig 对 topK chunk 进行检查, 判断是否找出的 topK 是否符合要求, 是否存在检索污染的情况
// 一个 EvalCheckConfig 对应一个问题
type EvalCheckConfig struct {
	ExpectSources   []string `json:"expected_sources"` // 相对文件路径集合, 表示期望 chunk 位置.
	MinScore        float64  `json:"min_score"`        // 要求的最低分数. 只有大于等于这个分数, 才认为与问题相关联
	QuestionId      string   `json:"question_id"`      // 问题 id
	QuestionContent string   `json:"question_content"` // 问题 内容
	ShouldAnswer    bool     `json:"should_answer"`    // 根据现有材料是否应当回答问题
}

// NewEvalCheckConfig 从 caseFilePath 中读取到 EvalCheckConfig 的配置, 并将 ExpectSources 中的路径转换为绝对路径
// 输入: caseFilePath, 应确保该文件存在, 且为 jsonl 格式, 每一条 json 都对应一个 EvalCheckConfig
// 输出: 从 caseFilePath 解析到的所有 EvalCheckConfig 配置. 若解析过程中出现严重错误, 则返回错误
func NewEvalCheckConfig(caseFilePath string) ([]*EvalCheckConfig, error) {
	var ret []*EvalCheckConfig
	f, err := os.Open(caseFilePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var evalConf EvalCheckConfig
		jsonLine := scanner.Text()
		err = json.Unmarshal([]byte(jsonLine), &evalConf)
		if err != nil {
			log.Println("json 反序列化失败:", err, ", 跳过本条 case. 问题 case:", jsonLine)
			continue
		}
		for i := range evalConf.ExpectSources {
			evalConf.ExpectSources[i], err = utils.ConvertToAbsPath(evalConf.ExpectSources[i])
			if err != nil {
				log.Println("转化为绝对路径失败:", err, ", 跳过本条路径. 问题文件路径:", evalConf.ExpectSources[i])
			}
		}
		ret = append(ret, &evalConf)
	}
	if scanner.Err() != nil {
		return nil, fmt.Errorf("读取文件 %s 出现错误: %w", caseFilePath, scanner.Err())
	}
	return ret, nil
}

// EvalCheckResult 检查报告. 主要包含以下几个方面
// 1. NotFindInExpectSources 有哪些 chunk 材料不在期望 source 集合中
// 2. ScoreTooLow 有哪些 chunk 材料位于 source 集合中, 但是关联分数小于最低分数要求
// 3. Legal 有哪些 chunk 材料符合要求
type EvalCheckResult struct {
	NotFindInExpectSources []*SearchResult
	ScoreTooLow            []*SearchResult
	Legal                  []*SearchResult
	EvalShouldAnswer       bool
}

// EvalCheck 按期望 source 和最低分数对 topK 检索结果进行分类。
// 输入: `chunkList` 是当前问题的 topK 检索结果。
// 输出: 返回不在期望 source、分数过低、合法 chunk 和是否应该回答的检查结果。
// 示例: `result := checkConf.EvalCheck(searchResults)`。
func (checkConf *EvalCheckConfig) EvalCheck(chunkList []*SearchResult) *EvalCheckResult {
	var checkResult EvalCheckResult
	for _, chunk := range chunkList {
		chunkSourceFilePath, err := utils.ConvertToAbsPath(chunk.Chunk.SourceFile)
		if err != nil {
			log.Println("chunk 路径有误, 无法转换为绝对路径. 跳过本条chunk. err:", err)
			continue
		}
		if slices.Contains(checkConf.ExpectSources, chunkSourceFilePath) == false {
			checkResult.NotFindInExpectSources = append(checkResult.NotFindInExpectSources, chunk)
			continue
		}
		if chunk.Score < checkConf.MinScore {
			checkResult.ScoreTooLow = append(checkResult.ScoreTooLow, chunk)
			continue
		}
		checkResult.Legal = append(checkResult.Legal, chunk)
	}
	checkResult.EvalShouldAnswer = len(checkResult.Legal) > 0
	return &checkResult
}

// SumFromCheckResult 打印一次离线 Eval 的分类汇总和明细。
// 输入: `checkRes` 是当前问题的 Eval 检查结果。
// 输出: 将不在期望 source、分数过低、合法 chunk 和回答判断打印到标准输出。
// 示例: `result := checkConf.EvalCheck(results); result.SumFromCheckResult()`。
func (checkRes *EvalCheckResult) SumFromCheckResult() {
	if checkRes == nil {
		fmt.Println("Eval 检查结果为空")
		return
	}

	fmt.Println("Eval 检查结果:")
	fmt.Printf("  不在期望 source: %d\n", len(checkRes.NotFindInExpectSources))
	printEvalSearchResults("  不在期望 source 明细", checkRes.NotFindInExpectSources)
	fmt.Printf("  分数过低: %d\n", len(checkRes.ScoreTooLow))
	printEvalSearchResults("  分数过低明细", checkRes.ScoreTooLow)
	fmt.Printf("  合法 chunk: %d\n", len(checkRes.Legal))
	printEvalSearchResults("  合法 chunk 明细", checkRes.Legal)
	fmt.Printf("  EvalShouldAnswer: %t\n", checkRes.EvalShouldAnswer)
}

// printEvalSearchResults 打印 Eval 分类中的检索结果。
// 输入: `label` 是分类名称, `results` 是该分类对应的检索结果。
// 输出: 将每个结果的分数、来源和 chunk 索引打印到标准输出。
// 示例: `printEvalSearchResults("合法 chunk 明细", results)`。
func printEvalSearchResults(label string, results []*SearchResult) {
	if len(results) == 0 {
		return
	}
	fmt.Printf("%s:\n", label)
	for i, result := range results {
		if result == nil || result.Chunk == nil {
			fmt.Printf("    [%d] result 为空\n", i+1)
			continue
		}
		fmt.Printf(
			"    [%d] score=%.4f source=%s#chunk-%d\n",
			i+1,
			result.Score,
			result.Chunk.SourceFile,
			result.Chunk.ChunkIndex,
		)
	}
}
