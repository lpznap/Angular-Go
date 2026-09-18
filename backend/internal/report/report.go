package report

import (
	"bytes"
	"context"
	"dailyworknotes/internal/domain"
	"encoding/csv"
	"fmt"
	"html/template"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var tpl = template.Must(template.New("report").Funcs(template.FuncMap{"rich": func(s string) template.HTML { return template.HTML(domain.Sanitize(s)) }, "hours": func(m int32) string { return fmt.Sprintf("%dh %02dm", m/60, m%60) }}).Parse(`<!doctype html><html lang="en"><head><meta charset="utf-8"><style>@page{size:A4;margin:18mm}body{font-family:'Noto Sans Thai','Noto Sans',sans-serif;color:#22362d;font-size:12px;line-height:1.65}h1{font-size:28px}h2{font-size:18px;margin-bottom:4px}.meta{color:#65796c}article{border-top:1px solid #ced9d1;padding:16px 0;break-inside:avoid}ul{padding-left:22px}footer{margin-top:24px}pre{white-space:pre-wrap}p{overflow-wrap:anywhere}</style></head><body><p class="meta">DAILY WORK NOTES / WORK REPORT</p><h1>Your work, documented.</h1>{{range .Notes}}<article><p class="meta">{{.WorkDate}} · {{.Project}} · {{.Status}} · {{hours .Minutes}}</p><h2>{{.Title}}</h2><div>{{rich .Description}}</div><ul>{{range .Tasks}}<li>{{if .Done}}☑{{else}}☐{{end}} {{.Text}}</li>{{end}}</ul>{{if .Blockers}}<p><strong>Blockers</strong><br>{{.Blockers}}</p>{{end}}{{if .NextSteps}}<p><strong>Next steps</strong><br>{{.NextSteps}}</p>{{end}}</article>{{end}}<footer><strong>Total time: {{hours .Total}}</strong></footer></body></html>`))

func HTML(notes []domain.Note) ([]byte, error) {
	var b bytes.Buffer
	var total int32
	for _, n := range notes {
		total += n.Minutes
	}
	e := tpl.Execute(&b, struct {
		Notes []domain.Note
		Total int32
	}{notes, total})
	return b.Bytes(), e
}
func Cell(s string) string {
	trim := strings.TrimLeft(s, " \t\r\n")
	if trim != "" && strings.ContainsAny(trim[:1], "=+-@") || strings.HasPrefix(s, "\t") || strings.HasPrefix(s, "\r") {
		return "'" + s
	}
	return s
}
func CSV(notes []domain.Note) ([]byte, error) {
	var b bytes.Buffer
	b.WriteString("\xef\xbb\xbf")
	w := csv.NewWriter(&b)
	_ = w.Write([]string{"id", "work_date", "title", "project", "description", "tasks", "status", "priority", "tags", "minutes", "blockers", "next_steps", "created_at", "updated_at"})
	for _, n := range notes {
		tasks := []string{}
		for _, t := range n.Tasks {
			tasks = append(tasks, fmt.Sprintf("[%t] %s", t.Done, t.Text))
		}
		row := []string{n.ID, n.WorkDate, n.Title, n.Project, domain.Plain(n.Description), strings.Join(tasks, "; "), n.Status, n.Priority, strings.Join(n.Tags, ", "), strconv.Itoa(int(n.Minutes)), n.Blockers, n.NextSteps, n.CreatedAt.Format(time.RFC3339), n.UpdatedAt.Format(time.RFC3339)}
		for i := range row {
			row[i] = Cell(row[i])
		}
		if e := w.Write(row); e != nil {
			return nil, e
		}
	}
	w.Flush()
	return b.Bytes(), w.Error()
}
func PDF(ctx context.Context, url string, notes []domain.Note) ([]byte, error) {
	html, e := HTML(notes)
	if e != nil {
		return nil, e
	}
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	p, e := w.CreateFormFile("files", "index.html")
	if e != nil {
		return nil, e
	}
	_, _ = p.Write(html)
	_ = w.WriteField("printBackground", "true")
	_ = w.Close()
	req, e := http.NewRequestWithContext(ctx, "POST", url+"/forms/chromium/convert/html", &body)
	if e != nil {
		return nil, e
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	client := http.Client{Timeout: 60 * time.Second}
	resp, e := client.Do(req)
	if e != nil {
		return nil, e
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("PDF renderer returned %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 50<<20))
}
