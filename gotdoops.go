package main

import (
	"crypto/md5"
	"flag"
	"fmt"
	"github.com/deckarep/golang-set"
	"github.com/nfnt/resize"
	"image/jpeg"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

/*
  TODO:
        1.) optimize further, by looking at file size first then following through with md5.
        2.) run concurrently since the problem is embarassingly parallel
        3.) more fixes, minor tweaks to make more robust
*/

const thumbnailDir = "thumbs/"

var fileTypes = mapset.NewSetFromSlice([]interface{}{"png", "jpg", "jpeg", "ncf"})
var fileCorpus = make(map[string][]string)

func visit(path string, f os.FileInfo, err error) error {
	if !f.IsDir() {
		pieces := strings.Split(path, ".")
		ext := pieces[len(pieces)-1]

		if fileTypes.Contains(ext) {
			h := hashFile(path)

			item, ok := fileCorpus[h]
			if ok {
				item = append(item, path)
				fileCorpus[h] = item
			} else {
				fileCorpus[h] = []string{path}
			}
		}
	}
	return nil
}

func hashFile(path string) string {
	h := md5.New()
	f, err := os.Open(path)
	if err != nil {
		log.Fatal("Couldn't open file: ", err)
	}
	defer f.Close()

	_, err = io.Copy(h, f)
	if err != nil {
		log.Fatal("Couldn't copy file bytes over to hash: ", err)
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

//TODO: generate an html report to disk, then open the report in the browser
// each filepath should be a link to the file on disk file:///
func showDuplicates() {

	rows := make([]string, 0)
	counter := 0
	for k, v := range fileCorpus {
		if len(v) > 1 {
			//fmt.Println(k, v)
			rows = append(rows, fmt.Sprintf("<tr><td>%d</td><td><img class=\"img-thumbnail\" src=\"thumbs/%s.jpg\" width=\"100px\" height=\"70px\"/></td><td>%s</td></tr>", counter, k, strings.Join(Wrap(v, "<a href=\"#\">", "</a>"), "<br />")))

			//take the first of the duplicates and generate a thumbnail
			resizeImage(v[0], k)

			counter++
		}
	}

	finalHTML := strings.Replace(htmlTemplate, "{{DATA}}", strings.Join(rows, ""), -1)
	f, err := os.Create("report.html")
	if err != nil {
		log.Fatal("Couldn't create file: ", err)
	}
	defer f.Close()
	f.WriteString(finalHTML)
}

func Wrap(s []string, prefix, suffix string) []string {
	result := make([]string, len(s))

	for i, v := range s {
		result[i] = strings.Replace(prefix, "#", v, -1) + v + suffix
	}
	return result
}

func main() {

	flag.Parse()
	root := flag.Arg(0)
	err := filepath.Walk(root, visit)
	if err != nil {
		log.Println("Walk failed with err: ", err)
	}

	showDuplicates()

	exec.Command("open", "report.html").Start()
}

func resizeImage(f string, id string) {

	file, err := os.Open(f)
	if err != nil {
		log.Fatal(err)
	}

	// decode jpeg into image.Image
	img, err := jpeg.Decode(file)
	if err != nil {
		log.Fatal(err)
	}
	file.Close()

	// resize to width 1000 using Lanczos resampling
	// and preserve aspect ratio
	m := resize.Resize(100, 70, img, resize.Bilinear)

	out, err := os.Create(filepath.Join(thumbnailDir, id+".jpg"))
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	// write new image to file
	jpeg.Encode(out, m, nil)
}

var htmlTemplate = `
<!DOCTYPE html>
<html>
  <head>
    <title>Duplicate Images</title>
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <!-- Bootstrap -->
    <link href="http://netdna.bootstrapcdn.com/bootstrap/3.0.3/css/bootstrap.min.css" rel="stylesheet">

    <!-- HTML5 Shim and Respond.js IE8 support of HTML5 elements and media queries -->
    <!-- WARNING: Respond.js doesn't work if you view the page via file:// -->
    <!--[if lt IE 9]>
      <script src="https://oss.maxcdn.com/libs/html5shiv/3.7.0/html5shiv.js"></script>
      <script src="https://oss.maxcdn.com/libs/respond.js/1.3.0/respond.min.js"></script>
    <![endif]-->
  </head>
  <body>
    <h3>Found the following duplicate images:</h3>

    <!-- jQuery (necessary for Bootstrap's JavaScript plugins) -->
    <script src="https://code.jquery.com/jquery.js"></script>
    <!-- Include all compiled plugins (below), or include individual files as needed -->
    <script src="http://netdna.bootstrapcdn.com/bootstrap/3.0.3/js/bootstrap.min.js"></script>
    <table class="table table-striped">
        <thead>
          <tr>
            <th>#</th>
            <th>ID</th>
            <th>Files</th>
          </tr>
        </thead>
        <tbody>
          {{DATA}}
        </tbody>
      </table>
  </body>
</html>
`
