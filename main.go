// Ultimate Settings Panel — desktop shell.
//
// The panel itself is panel.html: one self-contained page with every setting, tool and
// command in it. This program's whole job is to put that page in a proper window instead
// of a browser tab, and to answer one question the page cannot answer on its own — "is
// there a newer version?" — because a page loaded from a data: URL cannot call GitHub.
//
// Two rules this file exists to keep:
//
//  1. Nothing is requested at startup. The update check runs only when someone clicks
//     the button. Opening the panel makes no network request at all, which is what the
//     README promises and what every other TechyGeeksHome app does.
//  2. Nothing is ever downloaded or installed automatically. The check compares two
//     version numbers and, at most, offers a link.
package main

import (
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/jchv/go-webview2"
)

// Version of this build. The About box reads it from here rather than hard-coding it in
// the HTML, so there is exactly one place to change at release time.
const Version = "8.0.2"

const (
	githubOwner = "techygeekshome"
	githubRepo  = "Ultimate-Settings-Panel"
	productURL  = "https://techygeekshome.info/ultimate-settings-panel/"
	webView2URL = "https://developer.microsoft.com/microsoft-edge/webview2/"
)

//go:embed panel.html
var panelHTML string

func main() {
	w := webview2.NewWithOptions(webview2.WebViewOptions{
		Debug:     false,
		AutoFocus: true,
		WindowOptions: webview2.WindowOptions{
			Title:  "Ultimate Settings Panel " + Version,
			Width:  1180,
			Height: 820,
			Center: true,
		},
	})
	if w == nil {
		// Almost always means the Edge WebView2 runtime is not installed. It ships with
		// Windows 11 and current Windows 10, but not on LTSC or freshly imaged machines,
		// so say what is wrong and where to get it rather than exiting silently.
		messageBox(
			"This app needs the Microsoft Edge WebView2 runtime, which wasn't found.\r\n\r\n"+
				"You can install it free from:\r\n"+webView2URL,
			"Ultimate Settings Panel")
		return
	}
	defer w.Destroy()

	// Hand the page its own version number so the About box and the update dialog agree
	// with the binary they are running inside.
	w.Init(`window.uspVersion = ` + jsString(Version) + `;`)

	// The page calls this, and only from the "Check for updates" button.
	w.Bind("uspCheckUpdate", checkForUpdate)

	// A data: URL rather than a local web server: no listening socket on the machine, and
	// nothing for a firewall or a security team to ask about. The page's clipboard helper
	// already falls back to document.execCommand when the async API is unavailable, which
	// is the only thing an opaque origin costs here.
	w.Navigate("data:text/html;base64," + base64.StdEncoding.EncodeToString([]byte(panelHTML)))
	w.Run()
}

// updateResult is the shape panel.html expects back. Field names are fixed by the page.
type updateResult struct {
	OK      bool   `json:"ok"`
	Newer   bool   `json:"newer"`
	Latest  string `json:"latest,omitempty"`
	Current string `json:"current"`
	About   string `json:"about,omitempty"`
	URL     string `json:"url,omitempty"`
	Error   string `json:"error,omitempty"`
}

// checkForUpdate asks GitHub's public releases API whether a newer tag exists.
//
// It sends a user agent naming the app, its version and the site — GitHub rejects requests
// without one — and nothing else. No machine identifier, no usage data. It never downloads
// or installs anything: the most that happens is the page offers a link.
func checkForUpdate() updateResult {
	current := parseVersion(Version)
	res := updateResult{Current: Version}

	client := &http.Client{Timeout: 20 * time.Second}
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", githubOwner, githubRepo)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		res.Error = "Could not build the update request."
		return res
	}
	req.Header.Set("User-Agent", fmt.Sprintf("UltimateSettingsPanel/%s (+https://techygeekshome.info)", Version))
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		res.Error = "Couldn't reach GitHub. Check your connection and try again."
		return res
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// No release published yet. Not an error worth alarming anyone about.
		res.OK = true
		return res
	}
	if resp.StatusCode != http.StatusOK {
		res.Error = fmt.Sprintf("GitHub returned %d. Please try again later.", resp.StatusCode)
		return res
	}

	var rel struct {
		TagName string `json:"tag_name"`
		Name    string `json:"name"`
		Body    string `json:"body"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		res.Error = "Couldn't read the reply from GitHub."
		return res
	}

	latest := parseVersion(rel.TagName)
	if latest == nil {
		res.Error = "Couldn't read the latest version number."
		return res
	}

	res.OK = true
	res.Latest = strings.TrimPrefix(strings.TrimSpace(rel.TagName), "v")
	res.URL = firstNonEmpty(rel.HTMLURL, productURL)
	res.About = firstLine(rel.Body)
	res.Newer = compare(latest, current) > 0
	return res
}

var versionPattern = regexp.MustCompile(`(\d+)(?:\.(\d+))?(?:\.(\d+))?(?:\.(\d+))?`)

// parseVersion pulls the first version-like run out of whatever it is given, so a GitHub
// tag of "v8.0.1", "8.0.1" or "8.0.1-beta" all read the same. Returns nil if there is no
// number in there at all — the caller treats that as "could not read it" rather than
// guessing, because guessing wrong means telling someone to download something they have.
func parseVersion(s string) []int {
	m := versionPattern.FindStringSubmatch(s)
	if m == nil {
		return nil
	}
	out := make([]int, 4)
	for i := 1; i <= 4; i++ {
		if m[i] == "" {
			continue
		}
		n, err := strconv.Atoi(m[i])
		if err != nil {
			// A digit run too long for int. Unknowable rather than zero.
			return nil
		}
		out[i-1] = n
	}
	return out
}

// compare returns >0 if a is newer than b. Component-wise, so 8.10.0 beats 8.9.0 — which
// string comparison gets wrong, and which is the bug every hand-rolled updater ships with.
func compare(a, b []int) int {
	for i := 0; i < 4; i++ {
		switch {
		case a[i] > b[i]:
			return 1
		case a[i] < b[i]:
			return -1
		}
	}
	return 0
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// firstLine keeps the release-notes blurb to something that fits the dialog.
func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		s = s[:i]
	}
	s = strings.TrimSpace(strings.TrimLeft(s, "#* "))
	if len(s) > 220 {
		s = strings.TrimSpace(s[:220]) + "…"
	}
	return s
}

func jsString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// messageBox is the one bit of Win32 needed: if WebView2 is missing there is no window to
// show a message in, so it has to come from the OS. syscall rather than a dependency,
// because this is the only call in the program that needs it.
func messageBox(text, caption string) {
	user32 := syscall.NewLazyDLL("user32.dll")
	proc := user32.NewProc("MessageBoxW")
	t, _ := syscall.UTF16PtrFromString(text)
	c, _ := syscall.UTF16PtrFromString(caption)
	const mbIconInformation = 0x40
	proc.Call(0, uintptr(unsafe.Pointer(t)), uintptr(unsafe.Pointer(c)), mbIconInformation)
}
