package main

import (
	"GopherAI/common/aihelper"
	"GopherAI/common/mysql"
	"GopherAI/common/rabbitmq"
	"GopherAI/common/rag"
	"GopherAI/common/redis"
	"GopherAI/config"
	"GopherAI/dao/message"
	"GopherAI/router"
	"context"
	"fmt"
	"log"

	"github.com/joho/godotenv"
)

func init() {
	err := godotenv.Load("Env.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

// 从数据库加载消息并初始化 AIHelperManager
func readDataFromDB() error {
	manager := aihelper.GetGlobalManager()
	// 从数据库读取所有消息
	msgs, err := message.GetAllMessages()
	if err != nil {
		return err
	}
	// 遍历数据库消息
	for i := range msgs {
		m := &msgs[i]
		helper, err := manager.GetOrCreateAIHelper(m.UserName, m.SessionID)
		if err != nil {
			log.Printf("[readDataFromDB] failed to create helper for user=%s session=%s: %v", m.UserName, m.SessionID, err)
			continue
		}
		log.Println("readDataFromDB init:  ", helper.SessionID)
		helper.AddMessage(m.Content, m.UserName, m.IsUser, false)
	}
	log.Println("AIHelperManager init success ")
	return nil
}

func main() {
	conf := config.GetConfig()
	host := conf.MainConfig.Host
	port := conf.MainConfig.Port
	//初始化mysql
	if err := mysql.InitMysql(); err != nil {
		log.Println("InitMysql error , " + err.Error())
		return
	}
	//初始化AIHelperManager
	err := readDataFromDB()
	if err != nil {
		return
	}

	// 索引 Info 目录知识库到 Qdrant
	ragSvc := rag.GetService()
	if ragSvc.Enabled() {
		if err := ragSvc.IndexFromInfo(context.Background()); err != nil {
			log.Printf("RAG index warning: %v", err)
		}
	} else {
		log.Println("RAG disabled: configure OLLAMA_EMBEDDING_MODEL and Qdrant in Env.env to enable")
	}

	//初始化redis
	redis.Init()
	log.Println("redis init success  ")
	rabbitmq.InitRabbitMQ()
	log.Println("rabbitmq init success  ")
	r := router.InitRouter()
	err = r.Run(fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		panic(err)
	}
}
