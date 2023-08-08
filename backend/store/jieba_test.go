package store_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/go-ego/gse"
	"github.com/yanyiwu/gojieba"
)

func TestGse(t *testing.T) {
	var (
		text = "I I I Idol.To be or not to be has tries trying try The, that's the question! 今天天气不错！你好世界👋 انا احب الموز. يعجبني ، هل يعجبك؟"
	)
	var seg1 gse.Segmenter
	seg1.LoadDict()

	s1 := seg1.CutTrim(text)

	fmt.Println("seg1 Cut: ", s1, s1[len(s1)-1])
	// seg1 Cut:  [to be   or   not to be ,   that's the question!]

	fmt.Println("seg1 Cut: ", seg1.CutTrim(text))

	fmt.Println("seg1 Cut: ", seg1.CutSearch(text))

}

func TestJieba(t *testing.T) {
	var s string
	var words []string
	use_hmm := true
	x := gojieba.NewJieba()
	defer x.Free()

	s = "I come to beijing tsing hua university. 我来到北京清华大学"
	words = x.CutAll(s)
	fmt.Println(s)
	fmt.Println("全模式:", strings.Join(words, "/"))

	words = x.Cut(s, use_hmm)
	fmt.Println(s)
	fmt.Println("精确模式:", strings.Join(words, "/"))
	s = "比特币"
	words = x.Cut(s, use_hmm)
	fmt.Println(s)
	fmt.Println("精确模式:", strings.Join(words, "/"))

	x.AddWord("比特币")
	// x.AddWordEx("比特币", 100000, "")
	s = "比特币"
	words = x.Cut(s, use_hmm)
	fmt.Println(s)
	fmt.Println("添加词典后,精确模式:", strings.Join(words, "/"))

	s = "他来到了网易杭研大厦"
	words = x.Cut(s, use_hmm)
	fmt.Println(s)
	fmt.Println("新词识别:", strings.Join(words, "/"))

	s = "小明硕士毕业于中国科学院计算所，后在日本京都大学深造"
	words = x.CutForSearch(s, use_hmm)
	fmt.Println(s)
	fmt.Println("搜索引擎模式:", strings.Join(words, "/"))

	s = "长春市长春药店"
	words = x.Tag(s)
	fmt.Println(s)
	fmt.Println("词性标注:", strings.Join(words, ","))

	s = "区块链"
	words = x.Tag(s)
	fmt.Println(s)
	fmt.Println("词性标注:", strings.Join(words, ","))

	s = "长江大桥 is good"
	words = x.CutForSearch(s, !use_hmm)
	fmt.Println(s)
	fmt.Println("搜索引擎模式:", strings.Join(words, "/"))

	wordinfos := x.Tokenize(s, gojieba.SearchMode, !use_hmm)
	fmt.Println(s)
	fmt.Println("Tokenize:(搜索引擎模式)", wordinfos)

	wordinfos = x.Tokenize(s, gojieba.DefaultMode, !use_hmm)
	fmt.Println(s)
	fmt.Println("Tokenize:(默认模式)", wordinfos)

	keywords := x.ExtractWithWeight(s, 5)
	fmt.Println("Extract:", keywords)
}
