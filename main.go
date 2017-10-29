package main

import (
	"flag"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"text/template"
	"time"

	"github.com/labstack/echo"
)

var (
	dataDirPath   string
	address       string
	disableStatic bool
)

func init() {
	flag.StringVar(&address, "address", ":5000", "Bind to a specific ADDRESS:PORT")
	flag.StringVar(&dataDirPath, "dir", "./data", "Path to upload directory")
	flag.BoolVar(&disableStatic, "no-static", false, "Disable serving uploaded file")
	flag.Parse()

	var err error
	dataDirPath, err = filepath.Abs(dataDirPath)
	if err != nil {
		log.Fatal(err)
	}

	if err := os.MkdirAll(dataDirPath, 0744); err != nil {
		log.Fatal(err)
	}
}

type Template struct {
	templates *template.Template
}

func (r *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return r.templates.ExecuteTemplate(w, name, data)
}

type FileInfo struct {
	Name      string
	UpdatedAt time.Time
}

func getFiles() ([]FileInfo, error) {
	files, err := ioutil.ReadDir(dataDirPath)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	fileInfos := make([]FileInfo, 0, len(files))
	for _, file := range files {
		f := FileInfo{file.Name(), file.ModTime()}
		fileInfos = append(fileInfos, f)
	}

	sort.Slice(fileInfos, func(i, j int) bool {
		return fileInfos[i].UpdatedAt.After(fileInfos[j].UpdatedAt)
	})

	return fileInfos, nil
}

func getIndex(c echo.Context) error {
	files, err := getFiles()
	if err != nil {
		return err
	}

	return c.Render(http.StatusOK, "index.html", map[string]interface{}{
		"Files": files,
	})
}

func postFile(c echo.Context) error {
	fh, err := c.FormFile("upload_file")
	var file multipart.File
	if err != nil {
		return err
	}

	file, err = fh.Open()
	if err != nil {
		return err
	}
	defer file.Close()
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}

	path := filepath.Join(dataDirPath, fh.Filename)
	image, err := os.Create(path)
	if err != nil {
		return nil
	}
	defer image.Close()
	_, err = image.Write(data)
	if err != nil {
		return err
	}

	return c.Redirect(http.StatusSeeOther, "./")
}

func deleteFile(c echo.Context) error {
	file := c.FormValue("filename")
	path := filepath.Join(dataDirPath, file)
	err := os.Remove(path)
	if err != nil {
		return err
	}

	return c.Redirect(http.StatusSeeOther, "./")
}

func main() {
	e := echo.New()

	t := &Template{
		templates: template.Must(template.ParseGlob("*.html")),
	}
	e.Renderer = t

	if !disableStatic {
		e.Static("/files", dataDirPath)
	}

	e.GET("/", getIndex)
	e.POST("/upload", postFile)
	e.POST("/delete", deleteFile)

	e.Start(address)
}
