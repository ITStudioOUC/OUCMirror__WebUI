package main

import "time"

type Localized map[string]string
type Link struct {
	Title Localized `json:"title"`
	URL   string    `json:"url"`
}
type RelatedLink struct {
	Title Localized `json:"title"`
	Kind  string    `json:"kind"`
	URL   string    `json:"url"`
}
type Download struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Category    string         `json:"category"`
	Description string         `json:"description"`
	Files       []DownloadFile `json:"files"`
	// Read old settings and submissions without losing existing download links.
	Version string `json:"version,omitempty"`
	URL     string `json:"url,omitempty"`
}
type DownloadFile struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}
type Settings struct {
	Name         Localized     `json:"name"`
	Description  Localized     `json:"description"`
	LogoLight    string        `json:"logoLight"`
	LogoDark     string        `json:"logoDark"`
	Favicon      string        `json:"favicon"`
	Avatar       string        `json:"avatar"`
	Email        string        `json:"email"`
	Address      Localized     `json:"address"`
	Links        []Link        `json:"links"`
	RelatedLinks []RelatedLink `json:"relatedLinks"`
	Downloads    []Download    `json:"downloads"`
}
type Announcement struct {
	Title     string `json:"title"`
	Body      string `json:"body"`
	Published bool   `json:"published"`
	Pinned    bool   `json:"pinned"`
	Date      string `json:"date"`
}
type MirrorMetadata struct {
	Name        string    `json:"name"`
	Description Localized `json:"description"`
	Help        string    `json:"help"`
	Category    string    `json:"category"`
	Visible     bool      `json:"visible"`
	Website     string    `json:"website"`
}
type Job struct {
	Name         string    `json:"name"`
	Worker       string    `json:"worker"`
	Status       string    `json:"status"`
	LastUpdate   time.Time `json:"last_update"`
	LastStarted  time.Time `json:"last_started"`
	LastEnded    time.Time `json:"last_ended"`
	NextSchedule time.Time `json:"next_schedule"`
	Upstream     string    `json:"upstream"`
	Size         string    `json:"size"`
	Error        string    `json:"error_msg"`
}
type PublicMirror struct {
	Name        string    `json:"name"`
	Description Localized `json:"description"`
	Help        string    `json:"help"`
	Category    string    `json:"category"`
	Status      string    `json:"status"`
	LastSuccess *int64    `json:"lastSuccess"`
	LastAttempt *int64    `json:"lastAttempt"`
	SyncStarted *int64    `json:"syncStarted"`
	NextSync    *int64    `json:"nextSync"`
	Size        *float64  `json:"size"`
	Website     string    `json:"website"`
	Upstream    string    `json:"upstream"`
	BrowseURL   string    `json:"browseUrl"`
}
type Worker struct {
	ID         string    `json:"id"`
	LastOnline time.Time `json:"last_online"`
}
type Record struct {
	ID        string `json:"id"`
	Data      any    `json:"data"`
	Revision  int    `json:"revision"`
	UpdatedAt int64  `json:"updatedAt"`
}
type Session struct {
	UserID   int64
	Username string
	Role     string
	CSRF     string
}

func initialSettings() Settings {
	return Settings{
		Name:        Localized{"zh-CN": "中国海洋大学软件镜像源", "en-US": "Ocean University of China Software Mirrors"},
		Description: Localized{"zh-CN": "中国海洋大学软件镜像源（MirrorOUC）由中国海洋大学爱特工作室承建。", "en-US": "MirrorOUC is built by IT Studio, Ocean University of China."},
		LogoLight:   "/assets/mirrorOUC_dark.png", LogoDark: "/assets/mirrorOUC.png", Favicon: "/assets/ouc-seal.png", Avatar: "/assets/ouc-seal.png", Email: "contact@itstudio.club",
		Address:      Localized{"zh-CN": "山东省青岛市中国海洋大学西海岸校区 · 信息南楼五八创新创业工坊", "en-US": "58 Innovation Workshop, Information South Building, West Coast Campus, Ocean University of China, Qingdao, Shandong"},
		RelatedLinks: initialRelatedLinks(),
		Links: []Link{
			{Localized{"zh-CN": "中国海洋大学", "en-US": "Ocean University of China"}, "https://www.ouc.edu.cn/"},
			{Localized{"zh-CN": "爱特工作室", "en-US": "IT Studio"}, "https://www.itouc.cn/"},
			{Localized{"zh-CN": "信息科学与工程学部", "en-US": "Faculty of Information Science and Engineering"}, "https://it.ouc.edu.cn/"},
			{Localized{"zh-CN": "爱特荣誉墙", "en-US": "IT Studio honors"}, "https://honor.itouc.cn/"},
			{Localized{"zh-CN": "GitOUC", "en-US": "GitOUC"}, "https://git.itouc.cn/"},
			{Localized{"zh-CN": "爱特博客网", "en-US": "IT Studio blog"}, "https://blog.itouc.cn/"},
		}, Downloads: []Download{},
	}
}

func initialRelatedLinks() []RelatedLink {
	return []RelatedLink{
		{Title: Localized{"zh-CN": "软件源使用帮助", "en-US": "Mirror usage guides"}, Kind: "help", URL: ""},
		{Title: Localized{"zh-CN": "中国海洋大学爱特工作室", "en-US": "IT Studio, Ocean University of China"}, Kind: "url", URL: "https://www.itouc.cn/"},
		{Title: Localized{"zh-CN": "中国海洋大学", "en-US": "Ocean University of China"}, Kind: "url", URL: "https://www.ouc.edu.cn/"},
	}
}

// Older settings records get the existing sidebar entries until explicitly edited.
// An empty array is intentional and must stay empty.
func settingsDefaults(data any) any {
	if object, ok := data.(map[string]any); ok {
		if value, exists := object["relatedLinks"]; !exists || value == nil {
			object["relatedLinks"] = initialRelatedLinks()
		}
		var downloads []Download
		if decodeData(object["downloads"], &downloads) == nil {
			object["downloads"] = normalizeDownloads(downloads)
		}
	}
	return data
}
