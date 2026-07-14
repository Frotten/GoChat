package tools

import (
	"GopherAI/config"
	"encoding/json"
	"fmt"
	"go/format"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const maxFileReadBytes = 32 * 1024

func GoFmt(code string) (string, error) {
	buf, err := format.Source([]byte(code))
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

func fileBasePath() (string, error) {
	base := strings.TrimSpace(config.GetConfig().FileConfig.BasePath)
	if base == "" {
		return "", fmt.Errorf("文件目录未配置")
	}
	return filepath.Abs(base)
}

func fileWorkPath() (string, error) {
	base := strings.TrimSpace(config.GetConfig().FileConfig.WorkPath)
	if base == "" {
		return "", fmt.Errorf("文件工作目录未配置")
	}
	return filepath.Abs(base)
}

func resolveSafeFilePath(input string, PathFunc func() (string, error)) (string, error) {
	base, err := PathFunc()
	if err != nil {
		return "", err
	}
	input = normalizeWorkFilePath(input)
	if input == "" {
		return "", fmt.Errorf("file_path 不能为空")
	}
	var candidate string
	if filepath.IsAbs(input) {
		candidate = filepath.Clean(input)
	} else {
		candidate = filepath.Clean(filepath.Join(base, filepath.FromSlash(input)))
	}
	abs, err := filepath.Abs(candidate)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(base, abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("不允许访问工作目录外的文件")
	}
	return abs, nil
}

func normalizeWorkFilePath(input string) string {
	input = strings.TrimSpace(strings.ReplaceAll(input, "\\", "/"))
	for strings.HasPrefix(input, "./") {
		input = strings.TrimPrefix(input, "./")
	}
	if strings.HasPrefix(strings.ToLower(input), "work/") {
		return input[len("work/"):]
	}
	return input
}

func resolveReadableFilePath(input string) (string, error) {
	input = strings.TrimSpace(strings.ReplaceAll(input, "\\", "/"))
	if input == "" {
		return "", fmt.Errorf("file_path 不能为空")
	}
	lower := strings.ToLower(input)
	if strings.HasPrefix(lower, "info/") {
		return resolveSafeFilePath(input, fileBasePath)
	}
	if path, err := resolveSafeFilePath(normalizeWorkFilePath(input), fileWorkPath); err == nil {
		if _, statErr := os.Stat(path); statErr == nil {
			return path, nil
		}
	}
	return resolveSafeFilePath(input, fileBasePath)
}

func walkSearchRoots() ([]struct {
	root    string
	relBase string
}, error) {
	work, err := fileWorkPath()
	if err != nil {
		return nil, err
	}
	roots := []struct {
		root    string
		relBase string
	}{
		{root: work, relBase: "work"},
	}
	base, err := fileBasePath()
	if err != nil {
		return nil, err
	}
	infoDir := filepath.Join(base, "Info")
	if st, statErr := os.Stat(infoDir); statErr == nil && st.IsDir() {
		roots = append(roots, struct {
			root    string
			relBase string
		}{root: infoDir, relBase: "Info"})
	}
	return roots, nil
}

func toolJSONResult(payload map[string]interface{}) string {
	data, err := json.Marshal(payload)
	if err != nil {
		return `{"success":false,"retry":false,"message":"结果序列化失败"}`
	}
	return string(data)
}

func toolNoRetry(message string) string {
	return toolJSONResult(map[string]interface{}{
		"success": false,
		"retry":   false,
		"message": message,
	})
}

func isSearchableTextFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".txt", ".md", ".json", ".yaml", ".yml", ".xml", ".csv", ".log", ".go", ".js", ".ts", ".vue", ".html", ".css":
		return true
	default:
		return ext == ""
	}
}

func SearchFile(Keyword string) (string, error) {
	keyword := strings.TrimSpace(Keyword)
	if keyword == "" {
		return toolNoRetry("keyword 不能为空"), nil
	}

	roots, err := walkSearchRoots()
	if err != nil {
		return toolNoRetry(err.Error()), nil
	}

	lowerKeyword := strings.ToLower(keyword)
	type fileItem struct {
		Name string `json:"name"`
		Path string `json:"path"`
	}
	matches := make([]fileItem, 0)
	seen := make(map[string]bool)

	for _, root := range roots {
		walkErr := filepath.Walk(root.root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				return nil
			}
			if !isSearchableTextFile(info.Name()) {
				return nil
			}

			rel, relErr := filepath.Rel(root.root, path)
			if relErr != nil {
				return nil
			}
			displayPath := filepath.ToSlash(filepath.Join(root.relBase, rel))

			nameMatched := strings.Contains(strings.ToLower(info.Name()), lowerKeyword)
			contentMatched := false
			if !nameMatched {
				content, readErr := os.ReadFile(path)
				if readErr == nil && strings.Contains(strings.ToLower(string(content)), lowerKeyword) {
					contentMatched = true
				}
			}
			if !nameMatched && !contentMatched {
				return nil
			}
			if seen[displayPath] {
				return nil
			}
			seen[displayPath] = true
			matches = append(matches, fileItem{
				Name: info.Name(),
				Path: displayPath,
			})
			return nil
		})
		if walkErr != nil {
			return toolNoRetry(fmt.Sprintf("搜索失败: %v", walkErr)), nil
		}
	}
	if len(matches) == 0 {
		return toolJSONResult(map[string]interface{}{
			"success": true,
			"files":   []fileItem{},
			"message": fmt.Sprintf("未找到包含关键词 %q 的文件，请勿猜测文件名，可更换关键词或告知用户", keyword),
			"retry":   false,
		}), nil
	}
	return toolJSONResult(map[string]interface{}{
		"success": true,
		"files":   matches,
		"message": "请从 files 中选择一项，将其 path 原样传给 read_file 或 edit_file",
		"retry":   false,
	}), nil
}

func ReadFile(filePath string) (string, error) {
	path, err := resolveReadableFilePath(filePath)
	if err != nil {
		return toolNoRetry(fmt.Sprintf("路径无效: %v。不要重试 read_file，请重新调用 search_files 获取真实路径", err)), nil
	}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return toolNoRetry(fmt.Sprintf("文件 %q 不存在。不要重试 read_file，请重新调用 search_files 或告知用户文件不存在", filePath)), nil
		}
		return toolNoRetry(fmt.Sprintf("无法访问文件: %v。不要重试 read_file", err)), nil
	}
	if info.IsDir() {
		return toolNoRetry("目标是目录而非文件。不要重试 read_file，请从 search_files 结果中选择文件 path"), nil
	}
	file, err := os.Open(path)
	if err != nil {
		return toolNoRetry(fmt.Sprintf("打开文件失败: %v。不要重试 read_file", err)), nil
	}
	defer file.Close()
	limited := io.LimitReader(file, maxFileReadBytes+1)
	contentBytes, err := io.ReadAll(limited)
	if err != nil {
		return toolNoRetry(fmt.Sprintf("读取文件失败: %v。不要重试 read_file", err)), nil
	}
	truncated := len(contentBytes) > maxFileReadBytes
	if truncated {
		contentBytes = contentBytes[:maxFileReadBytes]
	}
	content := string(contentBytes)
	if truncated {
		content += fmt.Sprintf("\n\n[内容已截断，仅返回前 %d 字节]", maxFileReadBytes)
	}
	return toolJSONResult(map[string]interface{}{
		"success": true,
		"path":    filepath.ToSlash(filePath),
		"content": content,
	}), nil
}

func CreateFile(Title string) (string, error) {
	title := strings.TrimSpace(Title)
	if title == "" {
		return toolNoRetry("title 不能为空"), nil
	}
	if strings.ContainsAny(title, `\/`) {
		return toolNoRetry("title 不能包含路径分隔符"), nil
	}
	base, err := fileWorkPath()
	if err != nil {
		return toolNoRetry(err.Error()), nil
	}
	newPath := filepath.Join(base, title)
	if st, err := os.Stat(newPath); err == nil {
		rel, _ := filepath.Rel(base, newPath)
		if st.Size() == 0 {
			return toolJSONResult(map[string]interface{}{
				"success": true,
				"path":    filepath.ToSlash(rel),
				"message": "文件已存在且为空，请使用 edit_file 写入内容",
			}), nil
		}
		return toolNoRetry("文件已存在且非空，请更换 title 或使用 edit_file 修改"), nil
	} else if !os.IsNotExist(err) {
		return toolNoRetry(fmt.Sprintf("无法访问文件系统: %v", err)), nil
	}
	file, err := os.Create(newPath)
	if err != nil {
		return toolNoRetry(fmt.Sprintf("创建文件失败: %v", err)), nil
	}
	_ = file.Close()
	rel, _ := filepath.Rel(base, newPath)
	return toolJSONResult(map[string]interface{}{
		"success": true,
		"path":    filepath.ToSlash(rel),
	}), nil
}

func EditFile(FilePath string, Content string) (string, error) {
	path, err := resolveSafeFilePath(FilePath, fileWorkPath)
	if err != nil {
		return toolNoRetry(fmt.Sprintf("路径无效: %v。不要重试 edit_file，请重新调用 search_files 或 create_file 获取真实路径", err)), nil
	}
	err = os.WriteFile(path, []byte(Content), 0644)
	if err != nil {
		return toolNoRetry(fmt.Sprintf("写入文件失败: %v。不要重试 edit_file，请检查路径是否正确", err)), nil
	}
	return toolJSONResult(map[string]interface{}{
		"success": true,
		"path":    filepath.ToSlash(FilePath),
	}), nil
}

func GoFmtCode(Code string) (string, error) {
	formatted, err := GoFmt(Code)
	if err != nil {
		return toolNoRetry(fmt.Sprintf("代码格式化失败: %v。请确保只对 Go 代码使用 format_go_code 工具", err)), nil
	}
	return formatted, nil
}

func RenameFile(OldPath string, NewTitle string) (string, error) {
	oldPath, err := resolveSafeFilePath(OldPath, fileWorkPath)
	if err != nil {
		return toolNoRetry(fmt.Sprintf("旧路径无效: %v。不要重试 rename_file，请重新调用 search_files 获取真实路径", err)), nil
	}
	newTitle := strings.TrimSpace(NewTitle)
	if newTitle == "" {
		return toolNoRetry("new_title 不能为空"), nil
	}
	if strings.ContainsAny(newTitle, `\/`) {
		return toolNoRetry("new_title 不能包含路径分隔符"), nil
	}
	base, err := fileWorkPath()
	if err != nil {
		return toolNoRetry(err.Error()), nil
	}
	newPath := filepath.Join(base, newTitle)
	if st, err := os.Stat(newPath); err == nil {
		if st.IsDir() {
			return toolNoRetry("目标文件已存在且是目录，请更换 new_title 或使用 edit_file 修改内容"), nil
		}
		return toolNoRetry("目标文件已存在且非空，请更换 new_title 或使用 edit_file 修改"), nil
	} else if !os.IsNotExist(err) {
		return toolNoRetry(fmt.Sprintf("无法访问文件系统: %v", err)), nil
	}
	err = os.Rename(oldPath, newPath)
	if err != nil {
		return toolNoRetry(fmt.Sprintf("重命名文件失败: %v。不要重试 rename_file，请检查路径是否正确", err)), nil
	}
	rel, _ := filepath.Rel(base, newPath)
	return toolJSONResult(map[string]interface{}{
		"success": true,
		"path":    filepath.ToSlash(rel),
	}), nil
}
