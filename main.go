package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/adrg/frontmatter"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
)

type PostMetadata struct {
	Slug        string    `toml:"slug"`
	Title       string    `toml:"title"`
	Description string    `toml:"description"`
	Date        time.Time `toml:"date"`
	Language    string    `toml:"language"`
	Tags        []string  `toml:"tags"`
}

type PostData struct {
	Metadata PostMetadata
	Content  template.HTML
	Page     string
}

func main() {
	mdRenderer := goldmark.New(
		goldmark.WithExtensions(
			highlighting.NewHighlighting(
				highlighting.WithStyle("gruvbox"),
			),
		),
	)

	err := os.RemoveAll("public")
	if err != nil {
		log.Fatal(err)
	}

	err = os.MkdirAll("public", 0755)
	if err != nil {
		log.Fatal(err)
	}

	err = os.Mkdir("public/posts", 0755)
	if err != nil {
		log.Fatal(err)
	}

	err = copyFile("static/style.css", "public/style.css")
	if err != nil {
		log.Fatal("Error copying styles: ", err)
	}

	postsData, err := getAllpostData(mdRenderer)
	if err != nil {
		log.Fatal(err)
	}

	sort.Slice(postsData, func(i, j int) bool {
		return postsData[i].Metadata.Date.After(postsData[j].Metadata.Date)
	})

	err = ParsepostsMDToHTML(mdRenderer, postsData)
	if err != nil {
		log.Fatal(err)
	}

	err = ParseIndexTemplateToHTML(mdRenderer, postsData)
	if err != nil {
		log.Fatal(err)
	}

	err = ParseAboutPage()
	if err != nil {
		log.Fatal(err)
	}
}

func getAllpostData(mdRenderer goldmark.Markdown) ([]PostData, error) {
	var posts []PostData

	filenames, err := filepath.Glob("posts/*.md")
	if err != nil {
		log.Fatal(err)
	}

	for _, filename := range filenames {
		slug := strings.TrimPrefix(filename, "posts/")
		slug = strings.TrimSuffix(slug, ".md")

		// read markdown
		postMarkdown, err := Read(slug)
		if err != nil {
			return posts, fmt.Errorf("post Read error: %v", err)

		}

		rest, postData, err := getpostData(postMarkdown)
		if err != nil {
			return posts, err
		}

		// convert markdown to html
		var buf bytes.Buffer
		err = mdRenderer.Convert([]byte(rest), &buf)
		if err != nil {
			return posts, fmt.Errorf("Markdown to html convert error: %v", err)

		}

		postData.Content = template.HTML(buf.String())
		postData.Metadata.Slug = slug
		posts = append(posts, postData)
	}
	return posts, nil
}

func getpostData(postMarkdown io.Reader) (string, PostData, error) {
	var meta PostMetadata

	rest, err := frontmatter.Parse(postMarkdown, &meta)
	if err != nil {
		return "", PostData{}, fmt.Errorf("post parse error: %v", err)
	}

	postData := PostData{
		Metadata: meta,
	}

	return string(rest), postData, nil
}

func ParsepostsMDToHTML(mdRenderer goldmark.Markdown, postsData []PostData) error {
	tpl := template.Must(template.ParseFiles("template/post.html", "template/header.html", "template/footer.html"))

	for _, post := range postsData {
		// create .html file
		file, err := os.Create("public/posts/" + post.Metadata.Slug + ".html")
		if err != nil {
			return fmt.Errorf("HTML File creation error: %v", err)
		}

		defer file.Close()

		err = tpl.Execute(file, post)
		if err != nil {
			return fmt.Errorf("Tempalate Execute error: %v", err)

		}
	}
	return nil
}

type IndexData struct {
	Posts []PostData
	Page  string
}

func ParseIndexTemplateToHTML(mdRenderer goldmark.Markdown, postsData []PostData) error {
	tpl := template.Must(template.ParseFiles("template/index.html", "template/header.html", "template/footer.html"))

	file, err := os.Create("public/index.html")
	if err != nil {
		return fmt.Errorf("HTML File creation error: %v", err)
	}

	defer file.Close()

	indexData := IndexData{
		Posts: postsData,
		Page:  "index",
	}

	err = tpl.Execute(file, indexData)
	if err != nil {
		return fmt.Errorf("Tempalate Execute error: %v", err)
	}

	return nil
}

func ParseAboutPage() error {
	tpl := template.Must(template.ParseFiles(
		"template/about.html",
		"template/header.html",
		"template/footer.html",
	))

	file, err := os.Create("public/about.html")
	if err != nil {
		return err
	}

	defer file.Close()

	data := struct {
		Page  string
		Title string
	}{
		Page:  "about",
		Title: "About me",
	}

	return tpl.Execute(file, data)
}

func Read(slug string) (io.Reader, error) {
	f, err := os.Open("posts/" + slug + ".md")
	if err != nil {
		return nil, err
	}

	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(b), nil
}

func copyFile(srcPath, dstPath string) error {
	sourceFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}

	defer sourceFile.Close()

	destFile, err := os.Create(dstPath)
	if err != nil {
		return err
	}

	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	return nil
}
