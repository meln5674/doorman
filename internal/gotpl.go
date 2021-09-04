package internal

import (
	"os"
	gotpl "text/template"
)

func init() {
	TemplateFactories["gotpl"] = &GoTplFactory{}
	TemplateFactories[""] = TemplateFactories["gotpl"]
}

type GoTplTemplater struct {
	template *gotpl.Template
	path     string
}

func (g *GoTplTemplater) Template(vars interface{}) error {
	f, err := os.Create(g.path)
	if err != nil {
		return err
	}
	defer f.Close()
	return g.template.Execute(f, vars)
}

type GoTplFactory struct{}

func (g *GoTplFactory) Parse(template string, path string) (Templater, error) {
	tpl := GoTplTemplater{path: path, template: gotpl.New(path)}

	_, err := tpl.template.Parse(template)
	if err != nil {
		return nil, err
	}
	return &tpl, err
}
