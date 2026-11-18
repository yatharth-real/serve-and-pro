package server

import (
	"fmt"
	"html"
	"io"
	"os"
	"path"
	"sort"
	"strings"
	"time"
)

type fileEntry struct {
	name    string
	isDir   bool
	size    int64
	modTime time.Time
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// RenderDirectoryListing renders a clean, developer-focused directory listing.
func RenderDirectoryListing(w io.Writer, urlPath, diskPath string, entries []os.DirEntry) error {
	var list []fileEntry
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		list = append(list, fileEntry{
			name:    entry.Name(),
			isDir:   entry.IsDir(),
			size:    info.Size(),
			modTime: info.ModTime(),
		})
	}

	sort.Slice(list, func(i, j int) bool {
		if list[i].isDir != list[j].isDir {
			return list[i].isDir // directories first
		}
		return strings.ToLower(list[i].name) < strings.ToLower(list[j].name)
	})

	cleanURL := path.Clean("/" + strings.TrimPrefix(urlPath, "/"))
	if !strings.HasSuffix(cleanURL, "/") {
		cleanURL += "/"
	}

	title := html.EscapeString(cleanURL)

	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Index of %s</title>
  <style>
    :root {
      --bg: #0d1117;
      --card-bg: #161b22;
      --text: #e6edf3;
      --text-muted: #8b949e;
      --border: #30363d;
      --link: #58a6ff;
      --dir-color: #79c0ff;
    }
    @media (prefers-color-scheme: light) {
      :root {
        --bg: #f6f8fa;
        --card-bg: #ffffff;
        --text: #1f2328;
        --text-muted: #656d76;
        --border: #d0d7de;
        --link: #0969da;
        --dir-color: #0550ae;
      }
    }
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, monospace;
      background: var(--bg);
      color: var(--text);
      padding: 2.5rem 1.5rem;
      line-height: 1.5;
    }
    .container {
      max-width: 900px;
      margin: 0 auto;
      background: var(--card-bg);
      border: 1px solid var(--border);
      border-radius: 6px;
      overflow: hidden;
    }
    header {
      padding: 1rem 1.5rem;
      border-bottom: 1px solid var(--border);
      display: flex;
      justify-content: space-between;
      align-items: baseline;
    }
    header h1 {
      font-size: 1.1rem;
      font-weight: 600;
      font-family: monospace;
    }
    header .brand {
      font-size: 0.85rem;
      color: var(--text-muted);
      font-family: monospace;
    }
    table {
      width: 100%%;
      border-collapse: collapse;
      font-size: 0.9rem;
      font-family: monospace;
    }
    th {
      text-align: left;
      padding: 0.6rem 1.5rem;
      color: var(--text-muted);
      border-bottom: 1px solid var(--border);
      font-weight: 500;
    }
    td {
      padding: 0.5rem 1.5rem;
      border-bottom: 1px solid var(--border);
    }
    tr:last-child td { border-bottom: none; }
    tr:hover td { background: rgba(127, 127, 127, 0.05); }
    a {
      color: var(--link);
      text-decoration: none;
    }
    a:hover { text-decoration: underline; }
    .dir a {
      color: var(--dir-color);
      font-weight: 600;
    }
    .size, .date {
      color: var(--text-muted);
      white-space: nowrap;
    }
    .size { text-align: right; }
    .date { text-align: right; }
  </style>
</head>
<body>
  <div class="container">
    <header>
      <h1>Index of %s</h1>
      <span class="brand">Serve</span>
    </header>
    <table>
      <thead>
        <tr>
          <th>Name</th>
          <th style="text-align: right;">Size</th>
          <th style="text-align: right;">Modified</th>
        </tr>
      </thead>
      <tbody>
`, title, title)

	if cleanURL != "/" {
		parentURL := path.Dir(strings.TrimSuffix(cleanURL, "/"))
		if !strings.HasSuffix(parentURL, "/") {
			parentURL += "/"
		}
		fmt.Fprintf(w, `        <tr class="dir">
          <td><a href="%s">../</a></td>
          <td class="size">-</td>
          <td class="date">-</td>
        </tr>
`, html.EscapeString(parentURL))
	}

	for _, item := range list {
		escapedName := html.EscapeString(item.name)
		itemURL := path.Join(cleanURL, item.name)
		rowClass := "file"
		sizeStr := formatBytes(item.size)
		displayName := escapedName

		if item.isDir {
			itemURL += "/"
			displayName += "/"
			rowClass = "dir"
			sizeStr = "-"
		}

		fmt.Fprintf(w, `        <tr class="%s">
          <td><a href="%s">%s</a></td>
          <td class="size">%s</td>
          <td class="date">%s</td>
        </tr>
`, rowClass, html.EscapeString(itemURL), displayName, sizeStr, item.modTime.Format("2006-01-02 15:04:05"))
	}

	fmt.Fprintf(w, `      </tbody>
    </table>
  </div>
</body>
</html>
`)
	return nil
}
