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
	"os"
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
const Version = "8.0.4"

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

	// Every Run button on every card comes through here. Without this bind the page finds no
	// bridge and says so, which is what it did in 8.0.1 and 8.0.2.
	w.Bind("uspRun", runCommand)

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

// uriScheme matches something Windows already knows how to open by itself - ms-settings:,
// ms-windows-store:, microsoft-edge:, https: - while leaving a drive letter alone, because
// "C:\\..." is a path and not a scheme.
var uriScheme = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.\-]+:`)

// envVar matches a Windows %NAME% reference, so a command can be handed to ShellExecute with
// the variable already resolved. Anything sent to cmd keeps its %NAME% intact and lets cmd
// expand it, which is what makes a wildcard like %temp%\\* behave.
var envVar = regexp.MustCompile(`%([A-Za-z_][A-Za-z0-9_()]*)%`)

// consoleTools are the commands whose OUTPUT is the reason anyone pressed the button. Started
// on their own they print into a console that closes before it can be read, so they run under
// cmd /k and the window stays open. Shells are deliberately absent: powershell and wt already
// own their window.
var consoleTools = map[string]bool{
	"arp": true, "assoc": true, "chkdsk": true, "del": true, "dir": true, "dism": true,
	"driverquery": true, "getmac": true, "gpresult": true, "ipconfig": true, "manage-bde": true,
	"nbtstat": true, "net": true, "netsh": true, "netstat": true, "nslookup": true,
	"openfiles": true, "ping": true, "powercfg": true, "route": true, "set": true, "sfc": true,
	"systeminfo": true, "tasklist": true, "tracert": true, "ver": true, "vol": true,
	"whoami": true, "winget": true, "winmgmt": true, "wmic": true,
}

// runCommand launches one card from the panel.
//
// The page sends exactly the command printed on the card, so this has to cope with three
// shapes: a URI Windows opens for itself, a console tool whose output has to stay on screen,
// and everything else - GUI tools, .msc consoles, .cpl applets, Office switches.
//
// Nothing here is elevated and nothing is run in the background: this is the same launch a
// person would get by typing the command into Win+R, which is what the cards claim to be.
func runCommand(command string) error {
	command = strings.TrimSpace(command)
	if command == "" {
		return fmt.Errorf("there is nothing to run")
	}

	if uriScheme.MatchString(command) && !isDrivePath(command) {
		return shellExecute(command, "")
	}

	program, args := splitCommand(expandEnv(command))

	// cmd gets the ORIGINAL string: its own %NAME% expansion is what makes %temp%\\* work.
	if consoleTools[strings.TrimSuffix(strings.ToLower(program), ".exe")] {
		return shellExecute("cmd.exe", "/k "+command)
	}

	return shellExecute(program, args)
}

// isDrivePath tells "C:\\Windows" from "ms-settings:display" - a single letter before the colon
// is a drive, not a scheme.
func isDrivePath(s string) bool {
	return len(s) > 1 && s[1] == ':'
}

func expandEnv(s string) string {
	return envVar.ReplaceAllStringFunc(s, func(m string) string {
		if v, ok := os.LookupEnv(strings.Trim(m, "%")); ok {
			return v
		}
		return m
	})
}

// splitCommand separates the program from its arguments. A quoted program comes first; then a
// path that exists as written, because "C:\\Program Files\\...\\SCANPST.EXE" carries spaces and
// no quotes; and only then the first space.
func splitCommand(s string) (program, args string) {
	if strings.HasPrefix(s, `"`) {
		if end := strings.Index(s[1:], `"`); end >= 0 {
			return s[1 : end+1], strings.TrimSpace(s[end+2:])
		}
	}
	if _, err := os.Stat(s); err == nil {
		return s, ""
	}
	// A path carrying spaces with no switch after it is one file name, not a program plus
	// arguments - and it has to hold up on a machine where that file is not installed.
	if strings.ContainsAny(s, `\\/`) && !strings.Contains(s, " /") && !strings.Contains(s, " -") {
		return s, ""
	}
	if i := strings.IndexByte(s, ' '); i >= 0 {
		return s[:i], strings.TrimSpace(s[i+1:])
	}
	return s, ""
}

// shellExecute is the same call Win+R makes. The verb is left NULL so each file type gets its
// own default - "open" for an executable, but "cplopen" for a .cpl applet, which is why asking
// for "open" by name fails on half the Control Panel cards.
func shellExecute(file, params string) error {
	shell32 := syscall.NewLazyDLL("shell32.dll")
	proc := shell32.NewProc("ShellExecuteW")

	f, err := syscall.UTF16PtrFromString(file)
	if err != nil {
		return fmt.Errorf("that command could not be read")
	}
	var p *uint16
	if params != "" {
		if p, err = syscall.UTF16PtrFromString(params); err != nil {
			return fmt.Errorf("that command could not be read")
		}
	}

	const swShowNormal = 1
	r, _, _ := proc.Call(0, 0, uintptr(unsafe.Pointer(f)), uintptr(unsafe.Pointer(p)), 0, swShowNormal)

	// ShellExecute returns a value above 32 on success, and an error code below it. The three
	// worth naming are the ones a user can actually hit.
	if r > 32 {
		return nil
	}
	switch r {
	case 2, 3:
		return fmt.Errorf("Windows could not find %s on this machine", file)
	case 5:
		return fmt.Errorf("Windows refused to run %s - it may need administrator rights", file)
	case 31:
		return fmt.Errorf("nothing on this machine is set up to open %s", file)
	default:
		return fmt.Errorf("Windows could not start %s (error %d)", file, r)
	}
}
