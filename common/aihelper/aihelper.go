package aihelper

import (
	"GopherAI/common/rabbitmq"
	"GopherAI/common/rag"
	"GopherAI/model"
	"GopherAI/utils"
	"context"
	"fmt"
	"sync"

	"github.com/cloudwego/eino/schema"
)

// AIHelper AI助手结构体，包含消息历史和AI模型
type AIHelper struct {
	model    AIModel
	messages []*model.Message
	mu       sync.RWMutex
	//一个会话绑定一个AIHelper
	SessionID string
	saveFunc  func(*model.Message) (*model.Message, error)
}

// NewAIHelper 创建新的AIHelper实例
func NewAIHelper(model_ AIModel, SessionID string) *AIHelper {
	return &AIHelper{
		model:    model_,
		messages: make([]*model.Message, 0),
		//异步推送到消息队列中
		saveFunc: func(msg *model.Message) (*model.Message, error) {
			data := rabbitmq.GenerateMessageMQParam(msg.SessionID, msg.Content, msg.UserName, msg.IsUser)
			err := rabbitmq.RMQMessage.Publish(data)
			return msg, err
		},
		SessionID: SessionID,
	}
}

// AddMessage 添加消息到内存中并调用自定义存储函数
func (a *AIHelper) AddMessage(Content string, UserName string, IsUser bool, Save bool) {
	userMsg := model.Message{
		SessionID: a.SessionID,
		Content:   Content,
		UserName:  UserName,
		IsUser:    IsUser,
	}
	a.messages = append(a.messages, &userMsg)
	if Save {
		a.saveFunc(&userMsg)
	}
}

// SaveMessage 保存消息到数据库（通过回调函数避免循环依赖）
// 通过传入func，自己调用外部的保存函数，即可支持同步异步等多种策略
func (a *AIHelper) SetSaveFunc(saveFunc func(*model.Message) (*model.Message, error)) {
	a.saveFunc = saveFunc
}

// GetMessages 获取所有消息历史
func (a *AIHelper) GetMessages() []*model.Message {
	a.mu.RLock()
	defer a.mu.RUnlock()
	out := make([]*model.Message, len(a.messages))
	copy(out, a.messages)
	return out
}

func (a *AIHelper) buildMessagesWithRAG(ctx context.Context, userQuestion string) []*schema.Message {
	a.mu.RLock()
	messages := utils.ConvertToSchemaMessages(a.messages)
	a.mu.RUnlock()

	toolHint := &schema.Message{
		Role: schema.System,
		Content: "你是智能助手，可以帮助用户解决各方面的问题。涉及文件时严格按以下流程：" +
			"1) 先调用 search_files(keyword) 获取真实文件列表；" +
			"2) 从返回的 files 中选择一项；" +
			"3) 将选中项的 path 原样传给 read_file(file_path)。" +
			"禁止猜测文件名或路径。若 search_files 无结果或 read_file 返回 retry=false，不要重复调用 read_file，应更换关键词、告知用户或基于已有信息回答。",
	}

	contextText := rag.GetService().Retrieve(ctx, userQuestion)
	if contextText == "" {
		return append([]*schema.Message{toolHint}, messages...)
	}
	systemMsg := &schema.Message{
		Role: schema.System,
		Content: fmt.Sprintf(
			"你是智能助手。请优先根据以下参考资料回答用户问题；若资料中没有相关内容，请明确说明并基于常识谨慎回答，不要编造事实。"+
				"涉及文件时：先 search_files → 从列表选 path → read_file(file_path)；禁止猜测路径；read_file 失败且 retry=false 时不要重试。\n\n参考资料：\n%s",
			contextText,
		),
	}
	return append([]*schema.Message{systemMsg}, messages...)
}

// 同步生成
func (a *AIHelper) GenerateResponse(userName string, ctx context.Context, userQuestion string) (*model.Message, error) {

	//调用存储函数
	a.AddMessage(userQuestion, userName, true, true)

	messages := a.buildMessagesWithRAG(ctx, userQuestion)

	//调用模型生成回复
	schemaMsg, err := a.model.GenerateResponse(ctx, messages)
	if err != nil {
		return nil, err
	}

	//将schema.Message转化成model.Message
	modelMsg := utils.ConvertToModelMessage(a.SessionID, userName, schemaMsg)

	//调用存储函数
	a.AddMessage(modelMsg.Content, userName, false, true)

	return modelMsg, nil
}

// StreamResponse 流式生成
func (a *AIHelper) StreamResponse(userName string, ctx context.Context, cb StreamCallback, userQuestion string) (*model.Message, error) {

	//调用存储函数
	a.AddMessage(userQuestion, userName, true, true)
	messages := a.buildMessagesWithRAG(ctx, userQuestion)
	content, err := a.model.StreamResponse(ctx, messages, cb)
	if err != nil {
		return nil, err
	}
	//转化成model.Message
	modelMsg := &model.Message{
		SessionID: a.SessionID,
		UserName:  userName,
		Content:   content,
		IsUser:    false,
	}

	//调用存储函数
	a.AddMessage(modelMsg.Content, userName, false, true)

	return modelMsg, nil
}

// GetModelType 获取模型类型
func (a *AIHelper) GetModelType() string {
	return a.model.GetModelType()
}
