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

var promoteTemplate = `你是智能助手，用自然语言与用户对话。

【闲聊】问候、寒暄：直接回答，不调用任何工具。

【新建文件并写入内容】用户要创建/写一个带内容的新文件时：
1. 先创建空文件（只需文件名，如 note.txt，不要路径前缀）
2. 再写入内容（file_path 必须使用上一步返回的 path 原样，不要加 work/ 前缀）
3. 不要为此先搜索；新文件还不存在，搜索一定失败
4. 完成后用自然语言说明：文件名、是否成功、写入是否完成

【修改 work 目录下已有文件】仅当用户明确要改已有文件时：
1. 先搜索定位文件
2. 再覆盖写入；path 原样使用 search_files 返回的 path（例如 work/TempT）
3. 搜不到则告知用户，不要编造路径

【读取文件】路径未知时：先搜索再读取，基于读取结果回答。

【Info 目录】优先用已有上下文回答，不要写入 Info 目录。

【回复风格】禁止在回复中出现 JSON、函数名、tool、parameters；代码用 markdown 代码块。

【提问】 当用户提问时，请一步一步进行分析，如果需要调用工具，请先分析问题，明确需要调用的工具和参数，再调用工具。不要直接调用工具。`

// AIHelper AI助手结构体，包含消息历史和AI模型
type AIHelper struct {
	model    AIModel
	messages []*model.Message
	mu       sync.RWMutex
	//一个会话绑定一个AIHelper
	SessionID string
}

// NewAIHelper 创建新的AIHelper实例
func NewAIHelper(model_ AIModel, SessionID string) *AIHelper {
	return &AIHelper{
		model:     model_,
		messages:  make([]*model.Message, 0),
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
		_, _ = rabbitmq.SaveFunc(&userMsg)
	}
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
		Role:    schema.System,
		Content: promoteTemplate,
	}

	contextText := rag.GetService().Retrieve(ctx, userQuestion)
	if contextText == "" {
		return append([]*schema.Message{toolHint}, messages...)
	}
	systemMsg := &schema.Message{
		Role: schema.System,
		Content: fmt.Sprintf(promoteTemplate+"\n\n参考资料：\n%s",
			contextText,
		),
	}
	return append([]*schema.Message{systemMsg}, messages...)
}

func (a *AIHelper) GenerateResponse(userName string, ctx context.Context, userQuestion string) (*model.Message, error) {
	a.AddMessage(userQuestion, userName, true, true)
	messages := a.buildMessagesWithRAG(ctx, userQuestion)
	schemaMsg, err := a.model.GenerateResponse(ctx, messages)
	if err != nil {
		return nil, err
	}
	modelMsg := utils.ConvertToModelMessage(a.SessionID, userName, schemaMsg)
	a.AddMessage(modelMsg.Content, userName, false, true)
	return modelMsg, nil
}

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
