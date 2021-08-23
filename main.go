package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/signintech/gopdf"
)

const (
	Version       = "1.0"
	NameOfProgram = "resumeio2pdf"
	Copy          = "Copyright (c) 2021, Leonid Sopov <leonid@sopov.org>"
	CopyURL       = "https://github.com/sopov/resumeio2pdf/"

	resumePage = "https://resume.io/r/%s"
	resumeMeta = "https://ssr.resume.tools/meta/ssid-%s"
	resumeExt  = "png" // png, jpeg
	resumeIMG  = "https://ssr.resume.tools/to-image/ssid-%s-%d.%s?cache=%s&size=%d"
	resumeSize = 1800
	Timeout    = 60 * time.Second

	exitCodeMisuseArgs = 2
)

var (
	url         = flag.String("url", "", "link to resume of the format: https://resume.io/r/SecureID")
	sid         = flag.String("sid", "", "SecureID of resume")
	showVersion = flag.Bool("version", false, "show version")
	verbose     = flag.Bool("verbose", false, "show detail information")
	overWrite   = flag.Bool("y", false, "overwrite PDF file")
	pdfFileName = flag.String("pdf", "", "name of pdf file (default: SecureID + .pdf)")

	httpClient = &http.Client{Timeout: Timeout}
	reSID      = regexp.MustCompile(`^[[:alnum:]]+$`)
	reID       = regexp.MustCompile(`^\d+$`)
	reURL      = regexp.MustCompile(`^https://resume[.]io/r/([[:alnum:]]+)`)
	reIDURL    = regexp.MustCompile(`^https://resume[.]io/(?:app|api)/.*?/(\d+)`)
)

type metaLink struct {
	URL    string  `json:"url"`
	Left   float64 `json:"left"`
	Top    float64 `json:"top"`
	Height float64 `json:"height"`
	Width  float64 `json:"width"`
}

type metaViewPort struct {
	Height float64 `json:"height"`
	Width  float64 `json:"width"`
}

type metaPageInfo struct {
	Links    []metaLink   `json:"links"`
	ViewPort metaViewPort `json:"viewport"`
}

type metaInfo struct {
	Pages []metaPageInfo `json:"pages"`
}

func main() {
	if !readFlags() || *sid == "" {
		os.Exit(exitCodeMisuseArgs)
	}

	loggerf("SecureID: %s", *sid)
	loggerf("URL: %s", *url)
	loggerf("PDF: %s", *pdfFileName)

	meta, err := getMeta()
	if err != nil {
		log.Fatalln(err)
	}

	images, err := getResumeImages(len(meta.Pages))
	if err != nil {
		log.Fatalln(err)
	}

	err = generatePDF(meta, images)
	if err != nil {
		log.Fatalln(err)
	}

	cleanup(images)

	fmt.Printf("Resume stored to %s\n", *pdfFileName)
}

func cleanup(images []string) {
	for _, file := range images {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			continue
		}

		if err := os.Remove(file); err != nil {
			fmt.Printf("Error on remove `%s': %s", file, err.Error())
		} else {
			loggerf("Image `%s' successfully deleted.", file)
		}
	}
}

func generatePDF(info *metaInfo, images []string) error {
	pdf := gopdf.GoPdf{}

	logger("Start Generate PDF")

	pageSize := gopdf.Rect{
		W: info.Pages[0].ViewPort.Width,
		H: info.Pages[0].ViewPort.Height,
	}

	pdf.Start(gopdf.Config{PageSize: pageSize})

	for i, image := range images {
		loggerf("Add page #%d", i+1)

		pageSize := gopdf.Rect{
			W: info.Pages[i].ViewPort.Width,
			H: info.Pages[i].ViewPort.Height,
		}

		opt := gopdf.PageOption{
			PageSize: &pageSize,
		}
		pdf.AddPageWithOption(opt)

		err := pdf.Image(image, 0, 0, &pageSize)
		if err != nil {
			return err
		}

		for _, link := range info.Pages[i].Links {
			loggerf("Add link to %s", link.URL)

			x := link.Left
			y := pageSize.H - link.Top - link.Height
			pdf.AddExternalLink(link.URL, x, y, link.Width, link.Height)
		}
	}

	loggerf("Store PDF to `%s'", *pdfFileName)

	return pdf.WritePdf(*pdfFileName)
}

func getResumeImages(p int) (pages []string, err error) {
	if p < 1 {
		return nil, errors.New("required one or more pages")
	}

	for pID := 1; pID <= p; pID++ {
		pageFile := fmt.Sprintf("%s-%d.%s", *sid, pID, resumeExt)
		if _, err := os.Stat(pageFile); os.IsNotExist(err) {
			loggerf("Download image #%d/%d", pID, p)
			imgURL := fmt.Sprintf(resumeIMG, *sid, pID, resumeExt, time.Now().UTC().Format(time.RFC3339), resumeSize)

			if err := downloadPage(imgURL, pageFile); err != nil {
				return pages, err
			}
		}

		pages = append(pages, pageFile)
	}

	loggerf("Total %d pages", len(pages))

	return pages, nil
}

func downloadPage(imgURL, imgFile string) error {
	r, err := httpClient.Get(imgURL)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		return errors.New(r.Status)
	}

	file, err := os.Create(imgFile)
	if err != nil {
		return err
	}

	defer file.Close()

	_, err = io.Copy(file, r.Body)
	if err != nil {
		return err
	}

	return nil
}

func getJSON(url string, target interface{}) error {
	logger("Download meta information")

	r, err := httpClient.Get(url)
	if err != nil {
		return err
	}

	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("Can't download information from the site resume.io. Please, check URL.\n\nError: %s", r.Status)
	}

	decoder := json.NewDecoder(r.Body)

	err = decoder.Decode(&target)
	if err != nil {
		return err
	}

	return nil
}

func getMeta() (meta *metaInfo, err error) {
	metaURL := fmt.Sprintf(resumeMeta, *sid)
	meta = &metaInfo{}
	err = getJSON(metaURL, meta)

	return meta, err
}

func readFlags() bool {
	flag.Parse()

	if *showVersion {
		fmt.Printf("Version: %s\n", Version)

		return false
	}

	if !extractArg() {
		return false
	}

	if *sid == "" && *url == "" {
		usages()
		return false
	}

	if *sid != "" {
		if !reSID.MatchString(*sid) {
			fmt.Println("The ID must be as alphanumeric")
			return false
		}

		*url = fmt.Sprintf(resumePage, *sid)
	}

	if reIDURL.MatchString(*url) {
		usageID()
		return false
	}

	if !reURL.MatchString(*url) {
		msg := fmt.Sprintf("The URL must be in the format %s\n", resumePage)
		fmt.Printf(msg, "SecureID")

		return false
	}

	if *sid == "" {
		m := reURL.FindSubmatch([]byte(*url))
		*sid = string(m[1])
	}

	if *pdfFileName == "" {
		*pdfFileName = fmt.Sprintf("%s.pdf", *sid)
	}

	rePDF := regexp.MustCompile(`(?i)[.]pdf$`)
	if !rePDF.MatchString(*pdfFileName) {
		*pdfFileName = fmt.Sprintf("%s.pdf", *pdfFileName)
	}

	if _, err := os.Stat(*pdfFileName); !*overWrite && !os.IsNotExist(err) {
		fmt.Printf("File `%s' already exists.\n\nFor overwrite run with `-y' flag\n", *pdfFileName)

		return false
	}

	return true
}

func extractArg() bool {
	arg := flag.Arg(0)

	if arg == "" {
		return true
	}

	if reID.MatchString(arg) {
		usageID()
		return false
	}

	if reIDURL.MatchString(arg) {
		usageID()
		return false
	}

	if reSID.MatchString(arg) {
		*sid = arg
		return true
	}

	if reURL.MatchString(arg) {
		*url = arg
		return true
	}

	return true
}

func usages() {
	fileExec, err := os.Executable()
	if err == nil {
		fileExec = filepath.Base(fileExec)
	}

	if fileExec == "" {
		fileExec = NameOfProgram
	}

	fmt.Println("Syntax:")
	fmt.Println("  ", fileExec, "[options] [ID or URL]")

	fmt.Println()
	fmt.Println("Options:")
	flag.PrintDefaults()

	fmt.Println()
	fmt.Println(Copy)
	fmt.Println(CopyURL)
}

func usageID() {
	fmt.Println("Open in browser: https://resume.io/app/resumes")
	fmt.Println("Click on `...More' / `Share a link', and lunch with private URL.")
}

func logger(v ...interface{}) {
	if !*verbose {
		return
	}

	log.Println(v...)
}

func loggerf(format string, a ...interface{}) {
	if !*verbose {
		return
	}

	log.Printf(format, a...)
}
