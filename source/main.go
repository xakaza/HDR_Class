//go:build windows

package main

import (
	"crypto/rc4"
	"crypto/sha1"
	_ "embed"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	comdlg32 = syscall.NewLazyDLL("comdlg32.dll")
	comctl32 = syscall.NewLazyDLL("comctl32.dll")
	dwmapi   = syscall.NewLazyDLL("dwmapi.dll")
	uxtheme  = syscall.NewLazyDLL("uxtheme.dll")
)

var (
	pRegisterClassExW         = user32.NewProc("RegisterClassExW")
	pCreateWindowExW          = user32.NewProc("CreateWindowExW")
	pDefWindowProcW           = user32.NewProc("DefWindowProcW")
	pShowWindow               = user32.NewProc("ShowWindow")
	pUpdateWindow             = user32.NewProc("UpdateWindow")
	pGetMessageW              = user32.NewProc("GetMessageW")
	pTranslateMessage         = user32.NewProc("TranslateMessage")
	pDispatchMessageW         = user32.NewProc("DispatchMessageW")
	pPostQuitMessage          = user32.NewProc("PostQuitMessage")
	pSendMessageW             = user32.NewProc("SendMessageW")
	pPostMessageW             = user32.NewProc("PostMessageW")
	pSetWindowTextW           = user32.NewProc("SetWindowTextW")
	pGetWindowTextW           = user32.NewProc("GetWindowTextW")
	pGetWindowTextLengthW     = user32.NewProc("GetWindowTextLengthW")
	pEnableWindow             = user32.NewProc("EnableWindow")
	pSetWindowLongPtrW        = user32.NewProc("SetWindowLongPtrW")
	pSetFocus                 = user32.NewProc("SetFocus")
	pInvalidateRect           = user32.NewProc("InvalidateRect")
	pCreateIconFromResourceEx = user32.NewProc("CreateIconFromResourceEx")
	pDestroyIcon              = user32.NewProc("DestroyIcon")
	pLoadCursorW              = user32.NewProc("LoadCursorW")
	pSetBkColor               = gdi32.NewProc("SetBkColor")
	pSetTextColor             = gdi32.NewProc("SetTextColor")
	pSetBkMode                = gdi32.NewProc("SetBkMode")
	pCreateSolidBrush         = gdi32.NewProc("CreateSolidBrush")
	pCreatePen                = gdi32.NewProc("CreatePen")
	pCreateFontW              = gdi32.NewProc("CreateFontW")
	pSelectObject             = gdi32.NewProc("SelectObject")
	pCreateCompatibleDC       = gdi32.NewProc("CreateCompatibleDC")
	pCreateCompatibleBitmap   = gdi32.NewProc("CreateCompatibleBitmap")
	pDeleteDC                 = gdi32.NewProc("DeleteDC")
	pBitBlt                   = gdi32.NewProc("BitBlt")
	pRoundRect                = gdi32.NewProc("RoundRect")
	pDrawTextW                = user32.NewProc("DrawTextW")
	pFillRect                 = user32.NewProc("FillRect")
	pDrawIconEx               = user32.NewProc("DrawIconEx")
	pDeleteObject             = gdi32.NewProc("DeleteObject")
	pDragAcceptFiles          = shell32.NewProc("DragAcceptFiles")
	pDragQueryFileW           = shell32.NewProc("DragQueryFileW")
	pDragFinish               = shell32.NewProc("DragFinish")
	pSHBrowseForFolderW       = shell32.NewProc("SHBrowseForFolderW")
	pSHGetPathFromIDListW     = shell32.NewProc("SHGetPathFromIDListW")
	pGetOpenFileNameW         = comdlg32.NewProc("GetOpenFileNameW")
	pInitCommonControlsEx     = comctl32.NewProc("InitCommonControlsEx")
	pDwmSetWindowAttribute    = dwmapi.NewProc("DwmSetWindowAttribute")
	pSetWindowTheme           = uxtheme.NewProc("SetWindowTheme")
)

const (
	WS_OVERLAPPEDWINDOW  = 0x00CF0000
	WS_VISIBLE           = 0x10000000
	WS_CHILD             = 0x40000000
	WS_TABSTOP           = 0x00010000
	WS_BORDER            = 0x00800000
	WS_VSCROLL           = 0x00200000
	ES_AUTOHSCROLL       = 0x0080
	ES_MULTILINE         = 0x0004
	ES_AUTOVSCROLL       = 0x0040
	ES_READONLY          = 0x0800
	BS_PUSHBUTTON        = 0x00000000
	BS_AUTOCHECKBOX      = 0x00000003
	BS_OWNERDRAW         = 0x0000000B
	CBS_DROPDOWNLIST     = 0x0003
	CBS_OWNERDRAWFIXED   = 0x0010
	CBS_HASSTRINGS       = 0x0200
	SS_OWNERDRAW         = 0x0000000D
	SW_SHOW              = 5
	WM_DESTROY           = 0x0002
	WM_COMMAND           = 0x0111
	WM_DRAWITEM          = 0x002B
	WM_MEASUREITEM       = 0x002C
	WM_SETICON           = 0x0080
	WM_CTLCOLORSTATIC    = 0x0138
	WM_CTLCOLOREDIT      = 0x0133
	WM_DROPFILES         = 0x0233
	WM_SETFONT           = 0x0030
	WM_APP               = 0x8000
	BM_GETCHECK          = 0x00F0
	BM_SETCHECK          = 0x00F1
	BST_CHECKED          = 1
	CB_ADDSTRING         = 0x0143
	CB_SETCURSEL         = 0x014E
	CB_GETCURSEL         = 0x0147
	CB_GETLBTEXT         = 0x0148
	CB_GETLBTEXTLEN      = 0x0149
	EM_SETSEL            = 0x00B1
	EM_REPLACESEL        = 0x00C2
	PBM_SETRANGE32       = 0x0406
	PBM_SETPOS           = 0x0402
	PBM_SETBARCOLOR      = 0x0409
	PBM_SETBKCOLOR       = 0x2001
	COLOR_WINDOW         = 5
	IDC_ARROW            = 32512
	GWLP_ID              = -12
	OFN_FILEMUSTEXIST    = 0x00001000
	OFN_PATHMUSTEXIST    = 0x00000800
	OFN_EXPLORER         = 0x00080000
	BIF_RETURNONLYFSDIRS = 0x00000001
	BIF_NEWDIALOGSTYLE   = 0x00000040
	ICC_PROGRESS_CLASS   = 0x00000020
	TRANSPARENT          = 1
	PS_SOLID             = 0
	DT_CENTER            = 0x00000001
	DT_VCENTER           = 0x00000004
	DT_SINGLELINE        = 0x00000020
	ODS_SELECTED         = 0x0001
	ODS_DISABLED         = 0x0004
	ODS_FOCUS            = 0x0010
	ODS_COMBOBOXEDIT     = 0x1000
	ICON_SMALL           = 0
	ICON_BIG             = 1
	LR_DEFAULTCOLOR      = 0x0000
	SRCCOPY              = 0x00CC0020
)

const (
	idMode         = 100
	idLang         = 101
	idSource       = 102
	idBrowseSource = 103
	idOutput       = 104
	idBrowseOutput = 105
	idCheck1       = 106
	idCheck2       = 107
	idRun          = 108
	idCancel       = 109
	idLog          = 110
	idProgress     = 111
	idAppIcon      = 112
)

const (
	wmProgress = WM_APP + 1
	wmDone     = WM_APP + 2
	wmFlushLog = WM_APP + 3
)

type WNDCLASSEX struct {
	CbSize     uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   syscall.Handle
	Icon       syscall.Handle
	Cursor     syscall.Handle
	Background syscall.Handle
	MenuName   *uint16
	ClassName  *uint16
	IconSm     syscall.Handle
}

type POINT struct{ X, Y int32 }
type RECT struct{ Left, Top, Right, Bottom int32 }
type MSG struct {
	Hwnd           syscall.Handle
	Message        uint32
	WParam, LParam uintptr
	Time           uint32
	Pt             POINT
	LPrivate       uint32
}
type BROWSEINFO struct {
	Owner       syscall.Handle
	Root        uintptr
	DisplayName *uint16
	Title       *uint16
	Flags       uint32
	Callback    uintptr
	LParam      uintptr
	Image       int32
}
type OPENFILENAME struct {
	StructSize    uint32
	Owner         syscall.Handle
	Instance      syscall.Handle
	Filter        *uint16
	CustomFilter  *uint16
	MaxCustFilter uint32
	FilterIndex   uint32
	File          *uint16
	MaxFile       uint32
	FileTitle     *uint16
	MaxFileTitle  uint32
	InitialDir    *uint16
	Title         *uint16
	Flags         uint32
	FileOffset    uint16
	FileExtension uint16
	DefExt        *uint16
	CustData      uintptr
	Hook          uintptr
	TemplateName  *uint16
	Reserved      uintptr
	Reserved2     uint32
	FlagsEx       uint32
}
type INITCOMMONCONTROLSEX struct {
	Size uint32
	ICC  uint32
}
type MEASUREITEMSTRUCT struct {
	CtlType    uint32
	CtlID      uint32
	ItemID     uint32
	ItemWidth  uint32
	ItemHeight uint32
	ItemData   uintptr
}

type DRAWITEMSTRUCT struct {
	CtlType    uint32
	CtlID      uint32
	ItemID     uint32
	ItemAction uint32
	ItemState  uint32
	HwndItem   syscall.Handle
	HDC        syscall.Handle
	RcItem     RECT
	ItemData   uintptr
}

type archiveEntry struct {
	name   string
	path   string
	off    uint32
	length uint32
}

var (
	hMain                                                                                                          syscall.Handle
	hMode, hLang, hSource, hOutput, hBrowseSource, hBrowseOutput, hCheck1, hCheck2, hRun, hCancel, hLog, hProgress syscall.Handle
	hTitle, hSubtitle, hModeLabel, hLangLabel, hSourceLabel, hOutputLabel, hLogLabel                               syscall.Handle
	hFont, hTitleFont, hSmallFont, hProgressFont                                                                   syscall.Handle
	hBrush                                                                                                         syscall.Handle
	hIcon                                                                                                          syscall.Handle
	progressValue                                                                                                  atomic.Int32
	langEnglish                                                                                                    bool
	cancelFlag                                                                                                     atomic.Bool
	busy                                                                                                           atomic.Bool
	logMu                                                                                                          sync.Mutex
	pendingLogs                                                                                                    []string
	wndProcCallback                                                                                                uintptr
)

//go:embed hdrstudio.ico
var appIconIco []byte

func utf16(s string) *uint16 { p, _ := syscall.UTF16PtrFromString(s); return p }
func text(hwnd syscall.Handle) string {
	n, _, _ := pGetWindowTextLengthW.Call(uintptr(hwnd))
	if n == 0 {
		return ""
	}
	b := make([]uint16, n+1)
	pGetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&b[0])), n+1)
	return syscall.UTF16ToString(b)
}
func setText(hwnd syscall.Handle, s string) {
	pSetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(utf16(s))))
}
func send(hwnd syscall.Handle, msg uint32, w, l uintptr) uintptr {
	r, _, _ := pSendMessageW.Call(uintptr(hwnd), uintptr(msg), w, l)
	return r
}
func setFont(hwnd syscall.Handle) { send(hwnd, WM_SETFONT, uintptr(hFont), 1) }
func createControl(class, value string, style uint32, x, y, w, h int32, id int32) syscall.Handle {
	r, _, _ := pCreateWindowExW.Call(0, uintptr(unsafe.Pointer(utf16(class))), uintptr(unsafe.Pointer(utf16(value))), uintptr(style|WS_CHILD|WS_VISIBLE), uintptr(x), uintptr(y), uintptr(w), uintptr(h), uintptr(hMain), uintptr(id), 0, 0)
	hwnd := syscall.Handle(r)
	setFont(hwnd)
	if class == "Edit" || class == "ComboBox" {
		pSetWindowTheme.Call(uintptr(hwnd), uintptr(unsafe.Pointer(utf16("DarkMode_Explorer"))), 0)
	}
	return hwnd
}

func tr(tr, en string) string {
	if langEnglish {
		return en
	}
	return tr
}
func relabel() {
	if busy.Load() {
		return
	}
	setText(hMain, "HDR Studio")
	setText(hTitle, "HDR Studio")
	setText(hSubtitle, tr("HDR / SRC arşiv paketleme aracı", "HDR / SRC archive packaging utility"))
	setText(hModeLabel, tr("İşlem", "Operation"))
	setText(hLangLabel, tr("Dil", "Language"))
	setText(hSourceLabel, tr("Kaynak", "Source"))
	setText(hOutputLabel, tr("Çıktı", "Output"))
	setText(hLogLabel, tr("İşlem Günlüğü", "Activity Log"))
	if send(hMode, CB_GETCURSEL, 0, 0) == 0 {
		setText(hCheck1, tr("Mevcut HDR/SRC dosyalarını yedekle", "Back up existing HDR/SRC files"))
		setText(hCheck2, "")
	} else {
		setText(hCheck1, tr("Arşivleri çıkartmadan sonra koru", "Keep archives after extraction"))
		setText(hCheck2, tr("Var olan dosyaların üzerine yaz", "Overwrite existing files"))
	}
	setText(hBrowseSource, tr("Gözat", "Browse"))
	setText(hBrowseOutput, tr("Gözat", "Browse"))
	setText(hRun, tr("Başlat", "Start"))
	setText(hCancel, tr("İptal", "Cancel"))
}

func browseFolder(title string) string {
	var display [260]uint16
	bi := BROWSEINFO{Owner: hMain, DisplayName: &display[0], Title: utf16(title), Flags: BIF_RETURNONLYFSDIRS | BIF_NEWDIALOGSTYLE}
	pidl, _, _ := pSHBrowseForFolderW.Call(uintptr(unsafe.Pointer(&bi)))
	if pidl == 0 {
		return ""
	}
	var path [32768]uint16
	ok, _, _ := pSHGetPathFromIDListW.Call(pidl, uintptr(unsafe.Pointer(&path[0])))
	if ok == 0 {
		return ""
	}
	return syscall.UTF16ToString(path[:])
}
func browseHdr() string {
	var file [32768]uint16
	filter := syscall.StringToUTF16("HDR Archive (*.hdr)\x00*.hdr\x00All files (*.*)\x00*.*\x00\x00")
	ofn := OPENFILENAME{StructSize: uint32(unsafe.Sizeof(OPENFILENAME{})), Owner: hMain, Filter: &filter[0], File: &file[0], MaxFile: uint32(len(file)), Title: utf16(tr("HDR dosyası seç", "Select HDR file")), Flags: OFN_FILEMUSTEXIST | OFN_PATHMUSTEXIST | OFN_EXPLORER, DefExt: utf16("hdr")}
	ok, _, _ := pGetOpenFileNameW.Call(uintptr(unsafe.Pointer(&ofn)))
	if ok == 0 {
		return ""
	}
	return syscall.UTF16ToString(file[:])
}

func appendLogUI(s string) {
	if s == "" || hLog == 0 {
		return
	}
	old := text(hLog)
	if len(old) > 200000 {
		old = old[len(old)-150000:]
	}
	setText(hLog, old+s+"\r\n")
	send(hLog, EM_SETSEL, ^uintptr(0), ^uintptr(0))
}

// queueLog is safe from worker goroutines. Win32 controls are touched only
// from the UI thread when wmFlushLog is dispatched.
func queueLog(s string) {
	if s == "" {
		return
	}
	logMu.Lock()
	pendingLogs = append(pendingLogs, s)
	logMu.Unlock()
	if hMain != 0 {
		pPostMessageW.Call(uintptr(hMain), wmFlushLog, 0, 0)
	}
}

func flushLogsUI() {
	logMu.Lock()
	items := pendingLogs
	pendingLogs = nil
	logMu.Unlock()
	for _, s := range items {
		appendLogUI(s)
	}
}

func colorRef(r, g, b byte) uint32 {
	return uint32(r) | uint32(g)<<8 | uint32(b)<<16
}

func drawButton(dis *DRAWITEMSTRUCT) {
	id := dis.CtlID
	rect := dis.RcItem
	state := dis.ItemState
	disabled := (state & ODS_DISABLED) != 0
	pressed := (state & ODS_SELECTED) != 0
	var bg, border, fg uint32
	switch id {
	case idRun:
		bg = colorRef(46, 154, 255)
		border = colorRef(72, 176, 255)
		fg = colorRef(255, 255, 255)
		if pressed {
			bg = colorRef(32, 132, 227)
			border = colorRef(53, 149, 239)
		}
	case idCancel:
		bg = colorRef(56, 64, 78)
		border = colorRef(76, 86, 104)
		fg = colorRef(240, 244, 248)
		if pressed {
			bg = colorRef(45, 51, 62)
			border = colorRef(66, 74, 89)
		}
	default:
		bg = colorRef(40, 48, 61)
		border = colorRef(72, 84, 102)
		fg = colorRef(240, 244, 248)
		if pressed {
			bg = colorRef(31, 38, 49)
			border = colorRef(62, 74, 92)
		}
	}
	if disabled {
		bg = colorRef(42, 45, 52)
		border = colorRef(64, 68, 76)
		fg = colorRef(154, 160, 166)
	}
	brush, _, _ := pCreateSolidBrush.Call(uintptr(bg))
	pen, _, _ := pCreatePen.Call(PS_SOLID, 1, uintptr(border))
	oldBrush, _, _ := pSelectObject.Call(uintptr(dis.HDC), brush)
	oldPen, _, _ := pSelectObject.Call(uintptr(dis.HDC), pen)
	pRoundRect.Call(uintptr(dis.HDC), uintptr(rect.Left), uintptr(rect.Top), uintptr(rect.Right), uintptr(rect.Bottom), 12, 12)
	pSelectObject.Call(uintptr(dis.HDC), oldBrush)
	pSelectObject.Call(uintptr(dis.HDC), oldPen)
	pDeleteObject.Call(brush)
	pDeleteObject.Call(pen)
	pSetBkMode.Call(uintptr(dis.HDC), TRANSPARENT)
	pSetTextColor.Call(uintptr(dis.HDC), uintptr(fg))
	oldFont, _, _ := pSelectObject.Call(uintptr(dis.HDC), uintptr(hFont))
	defer pSelectObject.Call(uintptr(dis.HDC), oldFont)
	label := text(dis.HwndItem)
	buf := syscall.StringToUTF16(label)
	pDrawTextW.Call(uintptr(dis.HDC), uintptr(unsafe.Pointer(&buf[0])), ^uintptr(0), uintptr(unsafe.Pointer(&rect)), DT_CENTER|DT_VCENTER|DT_SINGLELINE)
}

func comboItemText(hwnd syscall.Handle, itemID uint32) string {
	if itemID == ^uint32(0) {
		idx := send(hwnd, CB_GETCURSEL, 0, 0)
		if idx == ^uintptr(0) {
			return ""
		}
		itemID = uint32(idx)
	}
	ln := send(hwnd, CB_GETLBTEXTLEN, uintptr(itemID), 0)
	if ln == ^uintptr(0) {
		return ""
	}
	buf := make([]uint16, int(ln)+1)
	send(hwnd, CB_GETLBTEXT, uintptr(itemID), uintptr(unsafe.Pointer(&buf[0])))
	return syscall.UTF16ToString(buf)
}

func drawCombo(dis *DRAWITEMSTRUCT) {
	r := dis.RcItem
	selected := (dis.ItemState & ODS_SELECTED) != 0
	bg := colorRef(28, 34, 43)
	fg := colorRef(232, 237, 244)
	if selected {
		bg = colorRef(42, 103, 178)
		fg = colorRef(255, 255, 255)
	}
	brush, _, _ := pCreateSolidBrush.Call(uintptr(bg))
	oldBrush, _, _ := pSelectObject.Call(uintptr(dis.HDC), brush)
	pen, _, _ := pCreatePen.Call(PS_SOLID, 1, uintptr(colorRef(62, 72, 87)))
	oldPen, _, _ := pSelectObject.Call(uintptr(dis.HDC), pen)
	pRoundRect.Call(uintptr(dis.HDC), uintptr(r.Left), uintptr(r.Top), uintptr(r.Right), uintptr(r.Bottom), 8, 8)
	pSelectObject.Call(uintptr(dis.HDC), oldBrush)
	pSelectObject.Call(uintptr(dis.HDC), oldPen)
	pDeleteObject.Call(brush)
	pDeleteObject.Call(pen)
	label := comboItemText(dis.HwndItem, dis.ItemID)
	pSetBkMode.Call(uintptr(dis.HDC), TRANSPARENT)
	pSetTextColor.Call(uintptr(dis.HDC), uintptr(fg))
	oldFont, _, _ := pSelectObject.Call(uintptr(dis.HDC), uintptr(hFont))
	defer pSelectObject.Call(uintptr(dis.HDC), oldFont)
	tr := RECT{r.Left + 12, r.Top, r.Right - 24, r.Bottom}
	buf := syscall.StringToUTF16(label)
	if len(buf) > 0 {
		pDrawTextW.Call(uintptr(dis.HDC), uintptr(unsafe.Pointer(&buf[0])), ^uintptr(0), uintptr(unsafe.Pointer(&tr)), DT_VCENTER|DT_SINGLELINE)
	}
}

func drawProgress(dis *DRAWITEMSTRUCT) {
	r := dis.RcItem
	width := int32(r.Right - r.Left)
	height := int32(r.Bottom - r.Top)
	if width <= 0 || height <= 0 {
		return
	}

	// Draw into an off-screen bitmap and copy the completed frame in one BitBlt.
	// This prevents the owner-draw control from flashing between background erase
	// and fill/text passes while progress updates arrive quickly.
	memDC, _, _ := pCreateCompatibleDC.Call(uintptr(dis.HDC))
	if memDC == 0 {
		return
	}
	defer pDeleteDC.Call(memDC)

	bmp, _, _ := pCreateCompatibleBitmap.Call(uintptr(dis.HDC), uintptr(width), uintptr(height))
	if bmp == 0 {
		return
	}
	oldBmp, _, _ := pSelectObject.Call(memDC, bmp)
	defer func() {
		pSelectObject.Call(memDC, oldBmp)
		pDeleteObject.Call(bmp)
	}()

	local := RECT{0, 0, width, height}
	pFillRect.Call(memDC, uintptr(unsafe.Pointer(&local)), uintptr(hBrush))

	track := colorRef(29, 35, 45)
	border := colorRef(58, 68, 82)
	fill := colorRef(48, 150, 255)

	brush, _, _ := pCreateSolidBrush.Call(uintptr(track))
	pen, _, _ := pCreatePen.Call(PS_SOLID, 1, uintptr(border))
	ob, _, _ := pSelectObject.Call(memDC, brush)
	op, _, _ := pSelectObject.Call(memDC, pen)
	pRoundRect.Call(memDC, 0, 0, uintptr(width), uintptr(height), 14, 14)
	pSelectObject.Call(memDC, ob)
	pSelectObject.Call(memDC, op)
	pDeleteObject.Call(brush)
	pDeleteObject.Call(pen)

	v := int(progressValue.Load())
	if v < 0 {
		v = 0
	}
	if v > 100 {
		v = 100
	}

	fillWidth := int(width-4) * v / 100
	if fillWidth > 0 {
		// Keep a stable rounded right edge once the bar is wide enough. For the
		// first few pixels use a tiny radius so the fill does not appear to pulse.
		fr := RECT{2, 2, 2 + int32(fillWidth), height - 2}
		radius := uintptr(10)
		if fillWidth < 12 {
			radius = 4
		}
		fb, _, _ := pCreateSolidBrush.Call(uintptr(fill))
		fp, _, _ := pCreatePen.Call(PS_SOLID, 0, uintptr(fill))
		ob, _, _ := pSelectObject.Call(memDC, fb)
		op, _, _ := pSelectObject.Call(memDC, fp)
		pRoundRect.Call(memDC, uintptr(fr.Left), uintptr(fr.Top), uintptr(fr.Right), uintptr(fr.Bottom), radius, radius)
		pSelectObject.Call(memDC, ob)
		pSelectObject.Call(memDC, op)
		pDeleteObject.Call(fb)
		pDeleteObject.Call(fp)
	}

	label := fmt.Sprintf("%d%%", v)
	buf := syscall.StringToUTF16(label)
	pSetBkMode.Call(memDC, TRANSPARENT)
	pSetTextColor.Call(memDC, uintptr(colorRef(248, 250, 253)))
	oldFont, _, _ := pSelectObject.Call(memDC, uintptr(hProgressFont))
	pDrawTextW.Call(memDC, uintptr(unsafe.Pointer(&buf[0])), ^uintptr(0), uintptr(unsafe.Pointer(&local)), DT_CENTER|DT_VCENTER|DT_SINGLELINE)
	pSelectObject.Call(memDC, oldFont)

	pBitBlt.Call(uintptr(dis.HDC), uintptr(r.Left), uintptr(r.Top), uintptr(width), uintptr(height), memDC, 0, 0, SRCCOPY)
}

func loadEmbeddedIcon() syscall.Handle {
	if len(appIconIco) < 22 {
		return 0
	}
	count := int(binary.LittleEndian.Uint16(appIconIco[4:6]))
	if count < 1 || len(appIconIco) < 6+count*16 {
		return 0
	}
	entry := 6 + (count-1)*16
	sz := binary.LittleEndian.Uint32(appIconIco[entry+8 : entry+12])
	off := binary.LittleEndian.Uint32(appIconIco[entry+12 : entry+16])
	if int(off+sz) > len(appIconIco) || sz == 0 {
		return 0
	}
	b := appIconIco[off : off+sz]
	r, _, _ := pCreateIconFromResourceEx.Call(uintptr(unsafe.Pointer(&b[0])), uintptr(len(b)), 1, 0x00030000, 0, 0, LR_DEFAULTCOLOR)
	return syscall.Handle(r)
}

func enableInputs(enable bool) {
	v := uintptr(0)
	if enable {
		v = 1
	}
	for _, h := range []syscall.Handle{hMode, hLang, hSource, hOutput, hCheck1, hCheck2, hRun} {
		pEnableWindow.Call(uintptr(h), v)
	}
	if enable {
		pEnableWindow.Call(uintptr(hCancel), 0)
	} else {
		pEnableWindow.Call(uintptr(hCancel), 1)
	}
}

func archiveKey() []byte {
	h := sha1.Sum([]byte("owsd9012%$1as!wpow1033b%!@%12"))
	// For this 128-bit RC4 case, CryptoAPI takes the first 128 bits of the SHA-1-derived key material.
	return append([]byte(nil), h[:16]...)
}
func cryptCount(v uint32) uint32 {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	c, _ := rc4.NewCipher(archiveKey())
	c.XORKeyStream(b, b)
	return binary.LittleEndian.Uint32(b)
}

func pack(folder string, backup bool, progress func(int), logf func(string)) error {
	abs, err := filepath.Abs(folder)
	if err != nil {
		return err
	}
	st, err := os.Stat(abs)
	if err != nil || !st.IsDir() {
		return errors.New("source folder not found")
	}
	base := filepath.Base(abs)
	hdrPath := filepath.Join(abs, base+".hdr")
	srcPath := filepath.Join(abs, base+".src")
	hdrTmp := hdrPath + ".tmp"
	srcTmp := srcPath + ".tmp"
	hdrBak := hdrPath + ".bak"
	srcBak := srcPath + ".bak"
	var entries []archiveEntry
	var total uint64
	err = filepath.Walk(abs, func(path string, info os.FileInfo, e error) error {
		if e != nil {
			return e
		}
		if cancelFlag.Load() {
			return errors.New("cancelled")
		}
		if info.IsDir() {
			return nil
		}
		if path == hdrPath || path == srcPath || path == hdrTmp || path == srcTmp || path == hdrBak || path == srcBak {
			return nil
		}
		rel, e := filepath.Rel(abs, path)
		if e != nil {
			return e
		}
		rel = strings.ReplaceAll(rel, "/", "\\")
		if len([]byte(rel)) > 65535 {
			return fmt.Errorf("filename too long: %s", rel)
		}
		if info.Size() > 0xffffffff {
			return fmt.Errorf("file too large: %s", rel)
		}
		if total+uint64(info.Size()) > 0xffffffff {
			return errors.New("archive exceeds 4 GiB")
		}
		entries = append(entries, archiveEntry{name: rel, path: path, length: uint32(info.Size())})
		total += uint64(info.Size())
		return nil
	})
	if err != nil {
		return err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].name < entries[j].name })
	var off uint64
	for i := range entries {
		entries[i].off = uint32(off)
		off += uint64(entries[i].length)
	}
	sf, err := os.Create(srcTmp)
	if err != nil {
		return err
	}
	defer sf.Close()
	buf := make([]byte, 1024*1024)
	for i, e := range entries {
		if cancelFlag.Load() {
			sf.Close()
			os.Remove(srcTmp)
			return errors.New("cancelled")
		}
		f, er := os.Open(e.path)
		if er != nil {
			return er
		}
		_, er = io.CopyBuffer(sf, io.LimitReader(f, int64(e.length)), buf)
		f.Close()
		if er != nil {
			return er
		}
		progress((i + 1) * 100 / max(1, len(entries)))
	}
	if err = sf.Close(); err != nil {
		return err
	}
	hf, err := os.Create(hdrTmp)
	if err != nil {
		return err
	}
	write := func(v interface{}) error { return binary.Write(hf, binary.LittleEndian, v) }
	if err = write(uint32(200000)); err != nil {
		return err
	}
	if err = write(cryptCount(uint32(len(entries)))); err != nil {
		return err
	}
	for _, e := range entries {
		nb := []byte(e.name)
		if err = write(uint16(len(nb))); err != nil {
			return err
		}
		if _, err = hf.Write(nb); err != nil {
			return err
		}
		if err = write(e.off); err != nil {
			return err
		}
		if err = write(e.length); err != nil {
			return err
		}
	}
	if err = hf.Close(); err != nil {
		return err
	}
	if backup {
		if _, e := os.Stat(hdrPath); e == nil {
			os.Remove(hdrBak)
			_ = copyFile(hdrPath, hdrBak)
		}
		if _, e := os.Stat(srcPath); e == nil {
			os.Remove(srcBak)
			_ = copyFile(srcPath, srcBak)
		}
	}
	os.Remove(srcPath)
	if err = os.Rename(srcTmp, srcPath); err != nil {
		return err
	}
	os.Remove(hdrPath)
	if err = os.Rename(hdrTmp, hdrPath); err != nil {
		return err
	}
	logf("Created: " + hdrPath)
	logf("Created: " + srcPath)
	return nil
}

func unpack(hdrPath, outDir string, keep, overwrite bool, progress func(int), logf func(string)) error {
	srcPath := strings.TrimSuffix(hdrPath, filepath.Ext(hdrPath)) + ".src"
	sf, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("matching SRC not found")
	}
	defer sf.Close()
	sfi, _ := sf.Stat()
	srcSize := uint64(sfi.Size())
	hf, err := os.Open(hdrPath)
	if err != nil {
		return err
	}
	defer hf.Close()
	var first uint32
	if binary.Read(hf, binary.LittleEndian, &first) != nil {
		return errors.New("truncated HDR")
	}
	encrypted := first >= 100000
	var count uint32
	if encrypted {
		if binary.Read(hf, binary.LittleEndian, &count) != nil {
			return errors.New("missing row count")
		}
		count = cryptCount(count)
	} else {
		count = first
	}
	if count > 5000000 {
		return errors.New("invalid row count")
	}
	entries := make([]archiveEntry, 0, count)
	for i := uint32(0); i < count; i++ {
		var n uint32
		if encrypted {
			var n16 uint16
			if binary.Read(hf, binary.LittleEndian, &n16) != nil {
				return errors.New("truncated index")
			}
			n = uint32(n16)
		} else {
			if binary.Read(hf, binary.LittleEndian, &n) != nil {
				return errors.New("truncated index")
			}
		}
		if n == 0 || n > 65535 {
			return errors.New("invalid filename length")
		}
		b := make([]byte, n)
		if _, err = io.ReadFull(hf, b); err != nil {
			return err
		}
		var off, l uint32
		if binary.Read(hf, binary.LittleEndian, &off) != nil || binary.Read(hf, binary.LittleEndian, &l) != nil {
			return errors.New("truncated index")
		}
		if uint64(off)+uint64(l) > srcSize {
			return fmt.Errorf("entry outside SRC: %s", string(b))
		}
		name := strings.ReplaceAll(string(b), "\\", string(os.PathSeparator))
		clean := filepath.Clean(name)
		if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
			return fmt.Errorf("unsafe path: %s", name)
		}
		entries = append(entries, archiveEntry{name: clean, off: off, length: l})
	}
	if outDir == "" {
		outDir = filepath.Join(filepath.Dir(hdrPath), strings.TrimSuffix(filepath.Base(hdrPath), filepath.Ext(hdrPath)))
	}
	if err = os.MkdirAll(outDir, 0755); err != nil {
		return err
	}
	buf := make([]byte, 1024*1024)
	for i, e := range entries {
		if cancelFlag.Load() {
			return errors.New("cancelled")
		}
		dst := filepath.Join(outDir, e.name)
		if !overwrite {
			if _, er := os.Stat(dst); er == nil {
				progress((i + 1) * 100 / max(1, len(entries)))
				continue
			}
		}
		if err = os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return err
		}
		if _, err = sf.Seek(int64(e.off), io.SeekStart); err != nil {
			return err
		}
		of, er := os.Create(dst)
		if er != nil {
			return er
		}
		_, er = io.CopyBuffer(of, io.LimitReader(sf, int64(e.length)), buf)
		of.Close()
		if er != nil {
			return er
		}
		progress((i + 1) * 100 / max(1, len(entries)))
	}
	if !keep {
		hf.Close()
		sf.Close()
		os.Remove(hdrPath)
		os.Remove(srcPath)
	}
	logf("Output: " + outDir)
	return nil
}
func copyFile(a, b string) error {
	in, e := os.Open(a)
	if e != nil {
		return e
	}
	defer in.Close()
	out, e := os.Create(b)
	if e != nil {
		return e
	}
	_, e = io.Copy(out, in)
	if ce := out.Close(); e == nil {
		e = ce
	}
	return e
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func beginOperation() {
	if busy.Swap(true) {
		return
	}
	cancelFlag.Store(false)
	enableInputs(false)
	progressValue.Store(0)
	pInvalidateRect.Call(uintptr(hProgress), 0, 0)
	appendLogUI("---")
	mode := send(hMode, CB_GETCURSEL, 0, 0)
	source := strings.TrimSpace(text(hSource))
	out := strings.TrimSpace(text(hOutput))
	c1 := send(hCheck1, BM_GETCHECK, 0, 0) == BST_CHECKED
	c2 := send(hCheck2, BM_GETCHECK, 0, 0) == BST_CHECKED
	go func() {
		progress := func(v int) { pPostMessageW.Call(uintptr(hMain), wmProgress, uintptr(v), 0) }
		logf := func(s string) { queueLog(s) }
		var err error
		if mode == 0 {
			err = pack(source, c1, progress, logf)
		} else {
			err = unpack(source, out, c1, c2, progress, logf)
		}
		if err != nil {
			queueLog(tr("Hata: ", "Error: ") + err.Error())
		} else {
			queueLog(tr("Tamamlandı.", "Completed."))
		}
		pPostMessageW.Call(uintptr(hMain), wmDone, 0, 0)
	}()
}

func modeChanged() {
	if send(hMode, CB_GETCURSEL, 0, 0) == 0 {
		setText(hSource, "")
		setText(hOutput, "")
		setText(hCheck1, tr("Mevcut HDR/SRC dosyalarını yedekle", "Back up existing HDR/SRC files"))
		send(hCheck1, BM_SETCHECK, BST_CHECKED, 0)
		setText(hCheck2, "")
		pEnableWindow.Call(uintptr(hCheck2), 0)
	} else {
		setText(hSource, "")
		setText(hOutput, "")
		setText(hCheck1, tr("Arşivleri çıkartmadan sonra koru", "Keep archives after extraction"))
		send(hCheck1, BM_SETCHECK, BST_CHECKED, 0)
		setText(hCheck2, tr("Var olan dosyaların üzerine yaz", "Overwrite existing files"))
		send(hCheck2, BM_SETCHECK, BST_CHECKED, 0)
		pEnableWindow.Call(uintptr(hCheck2), 1)
	}
}

func wndProc(hwnd syscall.Handle, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_COMMAND:
		id := int32(wParam & 0xffff)
		code := uint16((wParam >> 16) & 0xffff)
		switch id {
		case idMode:
			if code == 1 {
				modeChanged()
			}
		case idLang:
			if code == 1 {
				langEnglish = send(hLang, CB_GETCURSEL, 0, 0) == 1
				relabel()
				modeChanged()
			}
		case idBrowseSource:
			if send(hMode, CB_GETCURSEL, 0, 0) == 0 {
				if p := browseFolder(tr("Kaynak klasörü seç", "Select source folder")); p != "" {
					setText(hSource, p)
				}
			} else {
				if p := browseHdr(); p != "" {
					setText(hSource, p)
					setText(hOutput, filepath.Join(filepath.Dir(p), strings.TrimSuffix(filepath.Base(p), filepath.Ext(p))))
				}
			}
		case idBrowseOutput:
			if p := browseFolder(tr("Çıktı klasörü seç", "Select output folder")); p != "" {
				setText(hOutput, p)
			}
		case idRun:
			beginOperation()
		case idCancel:
			cancelFlag.Store(true)
		}
		return 0
	case wmProgress:
		next := int32(wParam)
		if next < 0 {
			next = 0
		}
		if next > 100 {
			next = 100
		}
		if progressValue.Swap(next) != next {
			// FALSE erase flag is essential for flicker-free owner drawing.
			pInvalidateRect.Call(uintptr(hProgress), 0, 0)
		}
		return 0
	case wmFlushLog:
		flushLogsUI()
		return 0
	case wmDone:
		flushLogsUI()
		busy.Store(false)
		enableInputs(true)
		relabel()
		return 0
	case WM_MEASUREITEM:
		mis := (*MEASUREITEMSTRUCT)(unsafe.Pointer(lParam))
		if mis != nil && (mis.CtlID == idMode || mis.CtlID == idLang) {
			mis.ItemHeight = 30
			return 1
		}
	case WM_DRAWITEM:
		dis := (*DRAWITEMSTRUCT)(unsafe.Pointer(lParam))
		if dis != nil {
			switch dis.CtlID {
			case idBrowseSource, idBrowseOutput, idRun, idCancel:
				drawButton(dis)
				return 1
			case idMode, idLang:
				drawCombo(dis)
				return 1
			case idProgress:
				drawProgress(dis)
				return 1
			case idAppIcon:
				if hIcon != 0 {
					pDrawIconEx.Call(uintptr(dis.HDC), 2, 2, uintptr(hIcon), 48, 48, 0, 0, 0x0003)
				}
				return 1
			}
		}
	case WM_DROPFILES:
		var path [32768]uint16
		pDragQueryFileW.Call(wParam, 0, uintptr(unsafe.Pointer(&path[0])), uintptr(len(path)))
		p := syscall.UTF16ToString(path[:])
		if fi, e := os.Stat(p); e == nil && fi.IsDir() {
			send(hMode, CB_SETCURSEL, 0, 0)
			modeChanged()
			setText(hSource, p)
		} else if strings.EqualFold(filepath.Ext(p), ".hdr") {
			send(hMode, CB_SETCURSEL, 1, 0)
			modeChanged()
			setText(hSource, p)
			setText(hOutput, filepath.Join(filepath.Dir(p), strings.TrimSuffix(filepath.Base(p), filepath.Ext(p))))
		}
		pDragFinish.Call(wParam)
		return 0
	case WM_CTLCOLORSTATIC:
		pSetBkColor.Call(wParam, 0x15120E)
		pSetTextColor.Call(wParam, 0xF5F0EA)
		return uintptr(hBrush)
	case WM_CTLCOLOREDIT:
		pSetBkColor.Call(wParam, 0x211D18)
		pSetTextColor.Call(wParam, 0xF5F0EA)
		return uintptr(hBrush)
	case WM_DESTROY:
		cancelFlag.Store(true)
		pPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := pDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
	return r
}

func main() {
	// HWNDs, dialogs and the message pump must remain on the same OS thread.
	// Without this, Go may migrate the goroutine after a GC/scheduler tick,
	// leaving the window apparently frozen even while idle.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	icc := INITCOMMONCONTROLSEX{Size: uint32(unsafe.Sizeof(INITCOMMONCONTROLSEX{})), ICC: ICC_PROGRESS_CLASS}
	pInitCommonControlsEx.Call(uintptr(unsafe.Pointer(&icc)))
	hinst, _, _ := kernel32.NewProc("GetModuleHandleW").Call(0)
	cur, _, _ := pLoadCursorW.Call(0, IDC_ARROW)
	hBrush = syscall.Handle(must(pCreateSolidBrush.Call(0x15120E)))
	fontHeight := int32(-18)
	titleFontHeight := int32(-30)
	smallFontHeight := int32(-16)
	hFont = syscall.Handle(must(pCreateFontW.Call(uintptr(uint32(fontHeight)), 0, 0, 0, 400, 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(utf16("Segoe UI"))))))
	hTitleFont = syscall.Handle(must(pCreateFontW.Call(uintptr(uint32(titleFontHeight)), 0, 0, 0, 600, 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(utf16("Segoe UI Semibold"))))))
	hSmallFont = syscall.Handle(must(pCreateFontW.Call(uintptr(uint32(smallFontHeight)), 0, 0, 0, 400, 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(utf16("Segoe UI"))))))
	progressFontHeight := int32(-16)
	hProgressFont = syscall.Handle(must(pCreateFontW.Call(uintptr(uint32(progressFontHeight)), 0, 0, 0, 700, 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(utf16("Segoe UI Semibold"))))))
	cls := utf16("HDRStudioNative")
	hIcon = loadEmbeddedIcon()
	wndProcCallback = syscall.NewCallback(wndProc)
	wc := WNDCLASSEX{CbSize: uint32(unsafe.Sizeof(WNDCLASSEX{})), WndProc: wndProcCallback, Instance: syscall.Handle(hinst), Icon: hIcon, Cursor: syscall.Handle(cur), Background: hBrush, ClassName: cls, IconSm: hIcon}
	pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	r, _, _ := pCreateWindowExW.Call(0, uintptr(unsafe.Pointer(cls)), uintptr(unsafe.Pointer(utf16("HDR Studio"))), WS_OVERLAPPEDWINDOW, 120, 100, 960, 760, 0, 0, hinst, 0)
	hMain = syscall.Handle(r)
	if hMain == 0 {
		return
	}
	if hIcon != 0 {
		send(hMain, WM_SETICON, ICON_BIG, uintptr(hIcon))
		send(hMain, WM_SETICON, ICON_SMALL, uintptr(hIcon))
	}
	dark := uint32(1)
	pDwmSetWindowAttribute.Call(uintptr(hMain), 20, uintptr(unsafe.Pointer(&dark)), unsafe.Sizeof(dark))
	pDragAcceptFiles.Call(uintptr(hMain), 1)
	createControl("Static", "", SS_OWNERDRAW, 28, 18, 52, 52, idAppIcon)
	hTitle = createControl("Static", "HDR Studio", WS_VISIBLE, 92, 18, 320, 38, 1)
	send(hTitle, WM_SETFONT, uintptr(hTitleFont), 1)
	hSubtitle = createControl("Static", "", 0, 92, 52, 460, 22, 5)
	send(hSubtitle, WM_SETFONT, uintptr(hSmallFont), 1)
	hModeLabel = createControl("Static", "İşlem", 0, 28, 94, 120, 20, 6)
	send(hModeLabel, WM_SETFONT, uintptr(hSmallFont), 1)
	hMode = createControl("ComboBox", "", CBS_DROPDOWNLIST|CBS_OWNERDRAWFIXED|CBS_HASSTRINGS|WS_TABSTOP, 28, 116, 260, 260, idMode)
	send(hMode, CB_ADDSTRING, 0, uintptr(unsafe.Pointer(utf16("Pack / Paketle"))))
	send(hMode, CB_ADDSTRING, 0, uintptr(unsafe.Pointer(utf16("Unpack / Çıkart"))))
	send(hMode, CB_SETCURSEL, 0, 0)
	hLangLabel = createControl("Static", "Dil", 0, 728, 20, 80, 20, 7)
	send(hLangLabel, WM_SETFONT, uintptr(hSmallFont), 1)
	hLang = createControl("ComboBox", "", CBS_DROPDOWNLIST|CBS_OWNERDRAWFIXED|CBS_HASSTRINGS|WS_TABSTOP, 728, 43, 174, 240, idLang)
	send(hLang, CB_ADDSTRING, 0, uintptr(unsafe.Pointer(utf16("Türkçe"))))
	send(hLang, CB_ADDSTRING, 0, uintptr(unsafe.Pointer(utf16("English"))))
	send(hLang, CB_SETCURSEL, 0, 0)
	hSourceLabel = createControl("Static", "Kaynak", 0, 28, 166, 200, 22, 2)
	send(hSourceLabel, WM_SETFONT, uintptr(hSmallFont), 1)
	hSource = createControl("Edit", "", WS_BORDER|WS_TABSTOP|ES_AUTOHSCROLL, 28, 190, 724, 36, idSource)
	hBrowseSource = createControl("Button", "Gözat", BS_OWNERDRAW|WS_TABSTOP, 764, 190, 138, 36, idBrowseSource)
	hOutputLabel = createControl("Static", "Çıktı", 0, 28, 242, 200, 22, 3)
	send(hOutputLabel, WM_SETFONT, uintptr(hSmallFont), 1)
	hOutput = createControl("Edit", "", WS_BORDER|WS_TABSTOP|ES_AUTOHSCROLL, 28, 266, 724, 36, idOutput)
	hBrowseOutput = createControl("Button", "Gözat", BS_OWNERDRAW|WS_TABSTOP, 764, 266, 138, 36, idBrowseOutput)
	hCheck1 = createControl("Button", "", BS_AUTOCHECKBOX|WS_TABSTOP, 28, 322, 520, 28, idCheck1)
	send(hCheck1, BM_SETCHECK, BST_CHECKED, 0)
	hCheck2 = createControl("Button", "", BS_AUTOCHECKBOX|WS_TABSTOP, 28, 354, 520, 28, idCheck2)
	pEnableWindow.Call(uintptr(hCheck2), 0)
	hRun = createControl("Button", "Başlat", BS_OWNERDRAW|WS_TABSTOP, 28, 400, 178, 44, idRun)
	hCancel = createControl("Button", "İptal", BS_OWNERDRAW|WS_TABSTOP, 218, 400, 132, 44, idCancel)
	pEnableWindow.Call(uintptr(hCancel), 0)
	hProgress = createControl("Static", "", SS_OWNERDRAW, 28, 462, 874, 28, idProgress)
	progressValue.Store(0)
	hLogLabel = createControl("Static", "İşlem Günlüğü", 0, 28, 510, 180, 22, 4)
	send(hLogLabel, WM_SETFONT, uintptr(hSmallFont), 1)
	hLog = createControl("Edit", "", WS_BORDER|WS_VSCROLL|ES_MULTILINE|ES_AUTOVSCROLL|ES_READONLY, 28, 536, 874, 160, idLog)
	relabel()
	modeChanged()
	pShowWindow.Call(uintptr(hMain), SW_SHOW)
	pUpdateWindow.Call(uintptr(hMain))
	pSetFocus.Call(uintptr(hSource))
	var m MSG
	for {
		ret, _, _ := pGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(ret) <= 0 {
			break
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		pDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
	if hFont != 0 {
		pDeleteObject.Call(uintptr(hFont))
	}
	if hTitleFont != 0 {
		pDeleteObject.Call(uintptr(hTitleFont))
	}
	if hSmallFont != 0 {
		pDeleteObject.Call(uintptr(hSmallFont))
	}
	if hProgressFont != 0 {
		pDeleteObject.Call(uintptr(hProgressFont))
	}
	if hIcon != 0 {
		pDestroyIcon.Call(uintptr(hIcon))
	}
	if hBrush != 0 {
		pDeleteObject.Call(uintptr(hBrush))
	}
}
func must(r uintptr, _ uintptr, _ error) uintptr { return r }
