package main

import (
	"fmt"
	"bytes"
	"encoding/gob"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	zmq "github.com/pebbe/zmq4"
)

var dbPath = "/home/liuruoyu/Desktop/github/sakura/sender/tables.sqlite"

type ANSWER struct {
	ID        uint64 `gorm:"column:id;primary_key"`
	QuestionId uint64 `gorm:"column:question_id" sql:"type:integer"`
	AnswerId uint64 `gorm:"column:answer_id" sql:"type:integer"`
	Question string `gorm:"column:question" sql:"type:text"`
	Answer string `gorm:"column:answer" sql:"type:text"`
	Star uint64 `gorm:"column:star" sql:"type:integer"`
}

func (ANSWER) TableName() string {
	return "answer"
}

func (a *ANSWER) AfterCreate() (err error) {
	requester, err := zmq.NewSocket(zmq.REQ)
	if err != nil {
		return
	}
	defer requester.Close()

	requester.Connect("tcp://localhost:5555")
	var msg bytes.Buffer
	encoder := gob.NewEncoder(&msg)
	encoder.Encode(*a)
	requester.Send(msg.String(), 0)

	reply, err := requester.Recv(0)
	if err != nil {
		return err
	}
	fmt.Println(reply)
	return
}


func main() {
	db, err := gorm.Open("sqlite3", dbPath)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	tx := db.Begin()

	answer := ANSWER{
		QuestionId: 111111,
		AnswerId: 2222222222,
		Question: "长得丑怎么活?",
		Answer: "长得丑还不去死!!!",
		Star: 1000,
	}
	if err := tx.Create(&answer).Error; err != nil {
		tx.Rollback()
		panic(err)
	}

	tx.Commit()
}
