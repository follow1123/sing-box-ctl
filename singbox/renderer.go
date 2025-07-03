package singbox

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/follow1123/sing-box-ctl/logger"
)

type ConfRenderer struct {
	log  logger.Logger
	tmpl *template.Template
}

func NewConfRenderer(log logger.Logger, templateFile string) (*ConfRenderer, error) {
	tmpl := template.New("sing-box-config-tmpl").Funcs(template.FuncMap{
		"nodeFilter": nodeFilter,
		"isDirect":   isDirect,
	})
	data, err := os.ReadFile(templateFile)
	if err != nil {
		return nil, fmt.Errorf("read template file error: \n\t%w", err)
	}
	tmpl, err = tmpl.Parse(string(data))
	if err != nil {
		return nil, fmt.Errorf("parse template error: \n\t%w", err)
	}
	return &ConfRenderer{
		log:  log,
		tmpl: tmpl,
	}, nil
}

func (cg *ConfRenderer) Render(pc *PartialConfig, opts *Options) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := cg.tmpl.Execute(buf, TemplateData{
		Pc:   pc,
		Opts: opts,
	}); err != nil {
		return nil, fmt.Errorf("render template error: \n\t%w", err)
	}
	return buf.Bytes(), nil
}

func nodeFilter(nodeNames []string, keys string) []string {
	keyArr := strings.Split(keys, "|")
	result := make([]string, 0)
	for _, t := range nodeNames {
		for _, k := range keyArr {
			if strings.Contains(t, k) {
				result = append(result, t)
				break
			}
		}
	}
	return result
}

func isDirect(name string) bool {
	return strings.Contains(name, "直连")
}
