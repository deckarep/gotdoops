gotdoops
========
quick and dirty script that detects duplicate image files and generates an html report with a list of duplicates.  

intentions
==========
I built this for my wife who wanted a way to easily identify duplicate images on our growing collection of pictures.  Plus, it was an excuse to write something new in Go. =)

how it works
============
Running this utility on a directory of images will do the following:

* Recursively scan the directory looking for image files (currently only jpegs)
* Add each image file path to a map of slices with the filesize as the key looking for potential duplicates
* MD5 hash all files that have the same filesize looking for true duplicates
* Generate thumbnails of all found duplicates
* Generate an HTML report allowing for filtering on particular folder
* Other than scanning for files and generating a report this command line app *DOES NOT* attempt to delete, rename or move files around.  It is purely designed to yield a report for further analysis currently.

future improvements
===================
*  Enhance to run in parallell with Go's awesome goroutine/channels
*  Enhance to work on additional image types or even any file type
*  Modify to use Go's template library (should have started with this, but was lazy)
*  Enhance with command line options as necessary
*  Add unit-tests

usage
=====
```Go
//To install
go get github.com/deckarep/gotdoops

//Then open up a command line and run the following:
go run gotdoops.go [folder/with/images]
```

contributions
=============
Are absolutely welcome and encouraged!
