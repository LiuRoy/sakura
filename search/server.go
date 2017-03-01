package main

import (
	"fmt"
	"reflect"
	"strconv"
	"net/http"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/huichen/wukong/engine"
	"github.com/huichen/wukong/types"
	"github.com/gin-gonic/gin"
)

var (
	searcher = engine.Engine{}
	answerMap = make(map[uint64]*ZhiHuItem)

	dictFile = "/home/liuruoyu/Projects/src/github.com/huichen/wukong/data/dictionary.txt"
	stopTokenFile = "/home/liuruoyu/Projects/src/github.com/huichen/wukong/data/stop_tokens.txt"
	dbPath = "/home/liuruoyu/Desktop/github/sakura/spider/tables.sqlite"
)

type ANSWER struct {
	ID        uint64 `gorm:"column:id,primary_key"`
	QuestionId uint64 `gorm:"column:question_id" sql:"type:integer"`
	AnswerId uint64 `gorm:"column:answer_id" sql:"type:integer"`
	Question string `gorm:"column:question" sql:"type:text"`
	Answer string `gorm:"column:answer" sql:"type:text"`
	Star uint64 `gorm:"column:star" sql:"type:integer"`
}

func (ANSWER) TableName() string {
	return "answer"
}

type LABEL struct {
	ID        uint64 `gorm:"column:id,primary_key"`
	QuestionId uint64 `gorm:"column:question_id" sql:"type:integer"`
	Label string `gorm:"column:label" sql:"type:varchar(50)"`
}

func (LABEL) TableName() string {
	return "label"
}

type ZhiHuScoreFields struct {
	StarCount uint64
}

type ZhiHuItem struct {
	QuestionId uint64 `json:"question_id"`
	AnswerId uint64 `json:"answer_id"`
	Question string `json:"question"`
	Answer string `json:"answer"`
	Star uint64 `json:"star"`
	Labels []string `json:"labels"`
}

type ZhiHuScoringCriteria struct{
}

func (criteria ZhiHuScoringCriteria) Score(
	doc types.IndexedDocument, fields interface{}) []float32 {
	if reflect.TypeOf(fields) != reflect.TypeOf(ZhiHuScoreFields{}) {
		return []float32{}
	}

	wsf := fields.(ZhiHuScoreFields)
	output := make([]float32, 2)
	output[0] = float32(doc.BM25)
	output[1] = float32(wsf.StarCount)
	return output
}

func addAnswer() {
	db, err := gorm.Open("sqlite3", dbPath)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	var labels []LABEL
	db.Find(&labels)

	labelMap := make(map[uint64][]string)
	for _, label := range labels {
		if labelList, ok := labelMap[label.QuestionId]; ok {
			labelMap[label.QuestionId] = append(labelList, label.Label)
		} else {
			labelArray := []string{}
			labelArray = append(labelArray, label.Label)
			labelMap[label.QuestionId] = labelArray
		}
	}

	var offset uint64 = 0
	var step uint64 = 100
	var answers []ANSWER
	for {
		db.Offset(offset).Limit(step).Find(&answers)
		if len(answers) == 0 {
			break
		}

		for _, answer := range answers {
			answerMap[answer.AnswerId] = &ZhiHuItem{
				QuestionId: answer.QuestionId,
				AnswerId: answer.AnswerId,
				Question: answer.Question,
				Answer: answer.Answer,
				Star: answer.Star,
				Labels: labelMap[answer.QuestionId],
			}
			searcher.IndexDocument(answer.AnswerId, types.DocumentIndexData{
				Content: fmt.Sprintf("%s %s", answer.Question, answer.Answer),
				Fields: ZhiHuScoreFields{StarCount: answer.Star},
				Labels: labelMap[answer.QuestionId],
			}, false)
		}
		offset += step
	}
	searcher.FlushIndex()
}

func query(c *gin.Context) {
	keyword := c.DefaultQuery("key", "婚姻")
	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"result": "invalid offset"})
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"result": "invalid offset"})
	}

	output := searcher.Search(types.SearchRequest{
		Text: keyword,
		RankOptions: &types.RankOptions{
			ScoringCriteria: &ZhiHuScoringCriteria{},
			OutputOffset:    offset,
			MaxOutputs:      limit,
		},
	})

	docs := []*ZhiHuItem{}
	for _, doc := range output.Docs {
		answerId := doc.DocId
		docs = append(docs, answerMap[answerId])
	}
	c.JSON(http.StatusOK, gin.H{"total": output.NumDocs, "data": docs})
}

func main() {
	searcher.Init(types.EngineInitOptions{
		SegmenterDictionaries: dictFile,
		StopTokenFile:         stopTokenFile,
		IndexerInitOptions: &types.IndexerInitOptions{
			IndexType: types.FrequenciesIndex,
		},
		NumShards: 1,
	})

	addAnswer()


	router := gin.Default()
	router.GET("/zhihu", query)
	router.Run("127.0.0.1:8088")
}
