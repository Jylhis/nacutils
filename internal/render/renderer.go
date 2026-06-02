package render

import (
	"os"
	"strconv"
	"strings"

	"github.com/jylhis/nacutils/internal/design"
	"github.com/jylhis/nacutils/internal/envelope"
)

const ansiReset = "\x1b[0m"

const DisableDesignRendererEnv = "NACUTILS_DISABLE_DESIGN_RENDERER"

type Renderer interface {
	Header(string, bool) string
	Separator(string, bool) string
	Kind(envelope.Kind, bool) string
	KindText(envelope.Kind, string, bool) string
	Name() string
}

type styleRenderer struct {
	name          string
	headerCode    string
	separatorCode string
	kindCodes     map[envelope.Kind]string
}

func New() Renderer {
	if envEnabled(os.Getenv(DisableDesignRendererEnv)) {
		return legacyRenderer()
	}

	contract, err := design.Load()
	if err != nil {
		return legacyRenderer()
	}

	renderer, err := designRenderer(contract)
	if err != nil {
		return legacyRenderer()
	}

	return renderer
}

func (r styleRenderer) Header(s string, styled bool) string {
	return styleWithCode(r.headerCode, s, styled)
}

func (r styleRenderer) Separator(s string, styled bool) string {
	return styleWithCode(r.separatorCode, s, styled)
}

func (r styleRenderer) Kind(kind envelope.Kind, styled bool) string {
	return styleWithCode(r.kindCodes[kind], string(kind), styled)
}

func (r styleRenderer) KindText(kind envelope.Kind, s string, styled bool) string {
	return styleWithCode(r.kindCodes[kind], s, styled)
}

func (r styleRenderer) Name() string {
	return r.name
}

func legacyRenderer() Renderer {
	return styleRenderer{
		name:          "legacy",
		headerCode:    "1;36",
		separatorCode: "2",
		kindCodes: map[envelope.Kind]string{
			envelope.KindNote:             "34",
			envelope.KindStatus:           "32",
			envelope.KindAttn:             "1;33",
			envelope.KindHeartbeatSummary: "1;36",
		},
	}
}

func designRenderer(contract design.Contract) (Renderer, error) {
	headerCode, ok := contractCode(contract, contract.Palette["text-heading"], true)
	if !ok {
		return nil, os.ErrInvalid
	}
	separatorCode, ok := contractCode(contract, contract.Palette["text-faint"], false)
	if !ok {
		return nil, os.ErrInvalid
	}

	noteCode, ok := contractCode(contract, contract.Status["status-info"], false)
	if !ok {
		return nil, os.ErrInvalid
	}
	statusCode, ok := contractCode(contract, contract.Status["status-ok"], false)
	if !ok {
		return nil, os.ErrInvalid
	}
	attnCode, ok := contractCode(contract, contract.Status["status-warn"], false)
	if !ok {
		return nil, os.ErrInvalid
	}
	summaryCode, ok := contractCode(contract, contract.Palette["brand"], true)
	if !ok {
		return nil, os.ErrInvalid
	}

	return styleRenderer{
		name:          contract.Tag,
		headerCode:    headerCode,
		separatorCode: separatorCode,
		kindCodes: map[envelope.Kind]string{
			envelope.KindNote:             noteCode,
			envelope.KindStatus:           statusCode,
			envelope.KindAttn:             attnCode,
			envelope.KindHeartbeatSummary: summaryCode,
		},
	}, nil
}

func contractCode(contract design.Contract, tone design.Tone, bold bool) (string, bool) {
	slot, ok := ansiSlot(contract, tone)
	if !ok {
		return "", false
	}

	base := ansiSGR(slot)
	if bold {
		return "1;" + base, true
	}
	return base, true
}

func ansiSlot(contract design.Contract, tone design.Tone) (int, bool) {
	light := design.NormalizeHex(tone.Light)
	dark := design.NormalizeHex(tone.Dark)

	for idx, color := range contract.ANSI {
		if light != "" && light == design.NormalizeHex(color.Light) {
			return idx, true
		}
		if dark != "" && dark == design.NormalizeHex(color.Dark) {
			return idx, true
		}
	}

	return 0, false
}

func ansiSGR(slot int) string {
	if slot < 8 {
		return strconv.Itoa(30 + slot)
	}
	return strconv.Itoa(90 + (slot - 8))
}

func styleWithCode(code string, s string, styled bool) string {
	if !styled || code == "" {
		return s
	}
	return "\x1b[" + code + "m" + s + ansiReset
}

func envEnabled(value string) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	return value == "1" || value == "true" || value == "yes" || value == "on"
}
