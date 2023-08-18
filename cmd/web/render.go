package main

import (
	"embed"
	"fmt"
	"net/http"
	"strings"
	"text/template"
)

//informacoes para enviar para ass templates
type templateData struct {
	StringMap map[string]string
	IntMap map[string]int
	Float map[string]float32
	Data map[string]interface{}
	CSRFToken string
	Flash string
	Warning string
	Error string
	IsAuthenticated int
	API string
	CssVersion string
	StripSK string
	StripePK string
}

//funcoes para utilizar nas templates 
var functions = template.FuncMap{
	"formatCurrency": FormatCurrency,
}

func FormatCurrency(n int) string {
	f := float32(n/100)
	return fmt.Sprintf("$%.2f", f)
}

//go:embed templates
var templateFs embed.FS

func (app *application) addDefaultData(td *templateData, r *http.Request) *templateData {
	//adicionar rota default inserida em main.go
	td.API = app.config.api

	//adicionar SK do stripe para utilizar em qualquer template, secret e key inseridas em main.go
	td.StripSK = app.config.stripe.secret
	td.StripePK = app.config.stripe.key
	return td
}

func (app *application) renderTemplate(w http.ResponseWriter, r *http.Request, page string, td *templateData, partials ...string) error {
	var t *template.Template
	var err error
	templateToRender := fmt.Sprintf("templates/%s.page.gohtml", page)

	//verificar se a template existe em templateCache
	_, templateInMap := app.templateCahe[templateToRender]

	if app.config.env == "production" && templateInMap {
		//utilizar a template que esta em cache
		t = app.templateCahe[templateToRender]
	} else {
		//construir a template
		t, err = app.parseTemplate(partials, page, templateToRender)
		if err != nil {
			app.errorLog.Println(err)
			return err
		}
	}

	if td == nil {
		td = &templateData{}
	}

	td = app.addDefaultData(td, r)

	err = t.Execute(w, td)
	if err != nil {
		app.errorLog.Println(err)
		return err
	}

	return nil
}

func (app *application) parseTemplate(partials []string, page string, templateTorender string) (*template.Template,error) {
	var t *template.Template
	var err error

	// buid partials
	if len(partials) > 0 {
		for i, x := range partials {
			partials[i] = fmt.Sprintf("templates/%s.partial.gohtml", x)
		}
	}

	if len(partials) > 0 {
		t, err = template.New(fmt.Sprintf("%s.page.gohtml", page)).Funcs(functions).ParseFS(templateFs, "templates/base.layout.gohtml", strings.Join(partials, ","), templateTorender)
	} else {
		t, err = template.New(fmt.Sprintf("%s.page.gohtml", page)).Funcs(functions).ParseFS(templateFs, "templates/base.layout.gohtml", templateTorender)
	}

	if err != nil {
		app.errorLog.Println(err)
		return nil, err
	}

	//inserir a template que nao esta em cache 	no cache
	app.templateCahe[templateTorender] = t
	return t, nil
}