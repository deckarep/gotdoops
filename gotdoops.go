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
        1.) run concurrently since the problem is embarassingly parallel
        2.) more fixes, minor tweaks to make more robust
        3.) Use template library instead of dirty string/replace hacks (I was lazy)
*/

const thumbnailDir = "thumbs/"

var fileTypes = mapset.NewSetFromSlice([]interface{}{"jpg", "jpeg", "JPG", "JPEG"})
var fileSizeCorpus = make(map[int64][]string)
var fileHashCorpus = make(map[string][]string)
var directoriesWithDupes = mapset.NewSet()

func visit(path string, f os.FileInfo, err error) error {
	if !f.IsDir() {
		pieces := strings.Split(path, ".")
		ext := pieces[len(pieces)-1]

		if fileTypes.Contains(ext) {

			log.Println("Analyzing file size: ", path)

			fi, err := os.Open(path)
			if err != nil {
				panic("Could open file!")
			}

			defer fi.Close()

			in, err := fi.Stat()
			if err != nil {
				panic("Could get file info!")
			}

			item, ok := fileSizeCorpus[in.Size()]
			if ok {
				item = append(item, path)
				fileSizeCorpus[in.Size()] = item
			} else {
				fileSizeCorpus[in.Size()] = []string{path}
			}
		}
	}
	return nil
}

func hashString(s string) string {
	h := md5.New()
	h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum(nil))
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

func findPotentialDuplicates() {

	log.Println(fileSizeCorpus)

	for _, v := range fileSizeCorpus {
		if len(v) > 1 {
			for _, path := range v {
				h := hashFile(path)

				log.Println("Analyzing hashes: ", path)
				item, ok := fileHashCorpus[h]
				if ok {
					item = append(item, path)
					fileHashCorpus[h] = item
				} else {
					fileHashCorpus[h] = []string{path}
				}
			}
		}
	}
}

//TODO: generate an html report to disk, then open the report in the browser
// each filepath should be a link to the file on disk file:///
func processDuplicates() {

	rows := make([]string, 0)
	counter := 0
	for k, v := range fileHashCorpus {
		if len(v) > 1 {

			hashes := make([]string, len(v))
			for i, fp := range v {
				dir := filepath.Dir(fp)
				directoriesWithDupes.Add(dir)
				hashes[i] = "dupe" + hashString(dir)
			}

			rows = append(rows, fmt.Sprintf("<tr class=\"%s\" style=\"display:none;\"><td><img class=\"img-thumbnail\" src=\"thumbs/%s.jpg\" width=\"100px\" height=\"70px\"/></td><td>%s</td></tr>", strings.Join(hashes, " "), k, strings.Join(Wrap(v, "<a href=\"#\">", "</a>&nbsp;&nbsp;<a href=\"!\"><span class=\"glyphicon glyphicon-folder-open\"></a></span>"), "<br />")))

			//take the first of the duplicates and generate a thumbnail
			resizeImage(v[0], k)

			counter++
		}
	}

	log.Printf("Detected %d duplicates.", counter)

	dirs := make([]string, directoriesWithDupes.Cardinality())
	dirCounter := 0
	for dir := range directoriesWithDupes.Iter() {
		v, ok := dir.(string)
		if ok {
			dirs[dirCounter] = v
		}
		dirCounter++
	}

	finalHTML := strings.Replace(htmlTemplate, "{{directories}}", strings.Join(Wrap(dirs, "<option value=\"@\">", "</option>"), ""), -1)
	finalHTML = strings.Replace(finalHTML, "{{DATA}}", strings.Join(rows, ""), -1)

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
		m := strings.Trim(v, " ")
		sourceFolder := filepath.Join(filepath.Dir(m), ".DS_Store")
		result[i] = strings.Replace(strings.Replace(prefix, "#", m, -1), "@", hashString(m), -1) + m + strings.Replace(suffix, "!", sourceFolder, -1)
	}
	return result
}

func main() {

	if _, err := os.Stat(thumbnailDir); os.IsNotExist(err) {
		os.Mkdir(thumbnailDir, 0700)
	}

	flag.Parse()
	root := flag.Arg(0)
	err := filepath.Walk(root, visit)
	if err != nil {
		log.Println("Walk failed with err: ", err)
	}

	//finds by matching file sizes (stating the file, for speed)
	findPotentialDuplicates()

	//finds by actually md5 hasing (byte analysis)
	processDuplicates()

	log.Println(directoriesWithDupes)

	exec.Command("open", "report.html").Start()
}

func resizeImage(f string, id string) {

	destinationFilePath := filepath.Join(thumbnailDir, id+".jpg")

	//if the file doesn't already exist, generate it
	if _, err := os.Stat(destinationFilePath); os.IsNotExist(err) {

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

		out, err := os.Create(destinationFilePath)
		if err != nil {
			log.Fatal(err)
		}
		defer out.Close()

		// write new image to file
		jpeg.Encode(out, m, nil)
	}
}

var htmlTemplate = `
<!DOCTYPE html>
<html>
  <head>
    <title>Duplicate Images</title>
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <!-- Bootstrap -->
    <link href="https://netdna.bootstrapcdn.com/bootstrap/3.0.3/css/bootstrap.min.css" rel="stylesheet">

    <!-- HTML5 Shim and Respond.js IE8 support of HTML5 elements and media queries -->
    <!-- WARNING: Respond.js doesn't work if you view the page via file:// -->
    <!--[if lt IE 9]>
      <script src="https://oss.maxcdn.com/libs/html5shiv/3.7.0/html5shiv.js"></script>
      <script src="https://oss.maxcdn.com/libs/respond.js/1.3.0/respond.min.js"></script>
    <![endif]-->
  </head>
  <body>

    <!-- jQuery (necessary for Bootstrap's JavaScript plugins) -->
    <script src="https://code.jquery.com/jquery.js"></script>
    <!-- Include all compiled plugins (below), or include individual files as needed -->
    <script src="https://netdna.bootstrapcdn.com/bootstrap/3.0.3/js/bootstrap.min.js"></script>
	<div class="container-fluid">
		<div class="col-lg-10 col-lg-offset-1">
    	<h3>Found the following duplicate images:</h3>
		<hr>
			<div class="panel panel-default">
				<div class="panel-body">
					<select id="dirDropdown" class="form-control">
						{{directories}}
					</select>
				</div>
			</div>
			<div class="panel panel-default">
				<div class="panel-body">
					<table class="table table-striped">
						<thead>
						  <tr>
							<th>Preview</th>
							<th>Files</th>
						  </tr>
						</thead>
						<tbody>
						  {{DATA}}
						</tbody>
					</table>
				</div>
			</div>
		</div>
	</div>
		<script>
    	$(function(){
    		$('#dirDropdown').change(function(e){
    			showFolder(e.currentTarget.value);
    		});
		});

    	function showFolder(name){
    		$('tbody tr').hide();

    		var items = $(".dupe" + name);
			items.show();    		
    	};
    </script>
  </body>
</html>
`
