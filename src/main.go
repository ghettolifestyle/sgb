package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gomarkdown/markdown"
	"github.com/otiai10/copy"
)

var EDITOR = "/usr/bin/vi"
var USER_DIR, _ = os.UserHomeDir()
var WORKING_DIR = USER_DIR + "/Documents/blog"
var BACKUP_DIR = WORKING_DIR + "/bak"
var DRAFT_DIR = WORKING_DIR + "/drafts"
var TEMPLATE_DIR = WORKING_DIR + "/templates"
var OUT_DIR = WORKING_DIR + "/out"
var postDir = OUT_DIR + "/p"
var REMOTE_ROOT_DIR = "/var/static"

var SSH_USER = "root"
var SSH_HOST = "kmai.xyz"
var SSH_PORT = 22

func check(err error) {
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
}

func syncPosts() {
	rsyncArgs := OUT_DIR + "/* " + SSH_USER + "@" + SSH_HOST + ":" + REMOTE_ROOT_DIR + "/"

	cmd := exec.Command("bash", "-c", "rsync -e \"ssh -i $HOME/.ssh/id_ed25519\" -a "+rsyncArgs)
	err := cmd.Run()
	check(err)
}

func fetchPosts() {
	posts, _ := os.ReadDir(postDir)

	for _, post := range posts {
		os.RemoveAll(postDir + "/" + post.Name())
	}

	rsyncArgs := SSH_USER + "@" + SSH_HOST + ":" + REMOTE_ROOT_DIR + "/p/ " + DRAFT_DIR + "/"

	cmd := exec.Command("bash", "-c", "rsync -e \"ssh -i $HOME/.ssh/id_ed25519\" -a "+rsyncArgs)
	err := cmd.Run()
	check(err)

	cmd = exec.Command("bash", "-c", "ssh -i $HOME/.ssh/id_ed25519 "+SSH_USER+"@"+SSH_HOST+" rm -r /var/static/*")
	err = cmd.Run()
	check(err)

	posts, _ = os.ReadDir(DRAFT_DIR)
	for _, post := range posts {
		os.RemoveAll(DRAFT_DIR + "/" + post.Name() + "/index.html")
	}
}

func build_atom() {
	atom_head, err := os.ReadFile(TEMPLATE_DIR + "/head_atom.xml")

	if err != nil {
		panic(err)
	}

	os.WriteFile(OUT_DIR+"/atom.xml", atom_head, 0644)
}

func launchEditor(editor string, file string) {
	cmd := exec.Command(editor, file)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout

	err := cmd.Run()
	check(err)
}

func select_post(prompt string) string {
	posts, err := os.ReadDir(postDir)
	check(err)

	for index, post := range posts {
		fmt.Printf("[%d] %s\n", index, post.Name())
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print(prompt)
	buf, err := reader.ReadString('\n')
	check(err)

	selection, err := strconv.Atoi(strings.ReplaceAll(buf, "\n", ""))
	check(err)

	return postDir + "/" + posts[selection].Name()
}

func createPost(title string) {
	nonalphanumericRegexp := regexp.MustCompile(`\W`)
	multiUnderscoreRegexp := regexp.MustCompile(`[_]{2,}`)

	title = strings.ReplaceAll(title, "\n", "")
	date := strconv.FormatInt(time.Now().Unix(), 10)
	buf := []byte(strings.ReplaceAll(title, " ", "_"))
	formattedTitle := nonalphanumericRegexp.ReplaceAll(buf, []byte(""))
	formattedTitle = multiUnderscoreRegexp.ReplaceAll(formattedTitle, []byte("_"))
	fmt.Println(string(formattedTitle))

	postDir := DRAFT_DIR + "/" + string(formattedTitle)
	postFile := postDir + "/in.md"

	os.Mkdir(postDir, 0755)
	file, err := os.OpenFile(
		postFile,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644)
	check(err)

	defer file.Close()

	file.WriteString("# " + title + "\n")
	file.WriteString("<span class=\"date\">" + date + "</span>\n")
	file.WriteString("\n")

	fmt.Println("created post " + title + " in draft dir")

	launchEditor(EDITOR, postFile)
}

func buildIndex() {
	// remove old index file
	os.RemoveAll(OUT_DIR + "/index.html")

	posts, err := os.ReadDir(postDir)
	check(err)

	sort.Slice(posts, func(i, j int) bool {
		fileInfoI, _ := posts[i].Info()
		fileInfoJ, _ := posts[j].Info()
		return fileInfoI.ModTime().After(fileInfoJ.ModTime())
	})

	indexFile, err := os.OpenFile(
		OUT_DIR+"/index.html",
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644)
	check(err)
	defer indexFile.Close()

	headHtml, err := os.ReadFile(TEMPLATE_DIR + "/head_index.html")
	check(err)
	indexFile.Write(headHtml)

	indexFile.WriteString("<p>some words on pages about stuff</p>\n")
	indexFile.WriteString("<h2>posts</h2>\n")
	indexFile.WriteString("<ul id=\"posts\">\n")

	for _, post := range posts {
		postName := post.Name()
		postFile, err := os.OpenFile(
			postDir+"/"+postName+"/in.md",
			os.O_RDONLY,
			0644)
		check(err)

		// read first line of markdown input containing the post title
		// remove leading hashtag using slice
		scanner := bufio.NewScanner(postFile)
		scanner.Scan()
		postTitle := scanner.Text()[2:]

		// read second line of markdown containing post date
		// in epoch format wrapped by span, e.g. <span class="date">1234567890</span>
		// use compiled regex to remove the span tags
		scanner.Scan()
		postDate := scanner.Text()
		dateRegexp := regexp.MustCompile(`[0-9]{10}`)
		// convert regex result to byte array to pass to Find(), then convert back to string
		regexpResult := string(dateRegexp.Find([]byte(postDate)))
		// convert epoch date string to int64 in decimal
		// goal is to pass this int64 to time.Unix() to get proper Time type
		regexpResultInt64, err := strconv.ParseInt(regexpResult, 10, 64)
		check(err)

		formattedDate := strings.ToLower(time.Unix(regexpResultInt64, 0).Format("Jan 02, 2006"))

		// set atime, mtime to epoch timestamp first generated when post was created
		// allows sorting when index file is built and post list is generated

		postFile.Close()

		indexFile.WriteString("<li><a href=\"p/" + postName + "\">" + postTitle + "</a>\n")
		indexFile.WriteString("<span class=\"date\">" + formattedDate + "</span></li>\n")
	}

	indexFile.WriteString("</ul>\n")

	footHtml, err := os.ReadFile(TEMPLATE_DIR + "/foot_index.html")
	check(err)
	indexFile.Write(footHtml)
}

func assemblePosts() {
	posts, err := os.ReadDir(DRAFT_DIR)
	check(err)

	for _, post := range posts {
		post := post.Name()

		os.RemoveAll(postDir + "/" + post + "/index.html")

		htmlOutput, err := os.OpenFile(
			DRAFT_DIR+"/"+post+"/index.html",
			os.O_APPEND|os.O_CREATE|os.O_WRONLY,
			0644)
		check(err)
		defer htmlOutput.Close()

		headHtml, err := os.ReadFile(
			TEMPLATE_DIR + "/head_post.html")
		check(err)
		htmlOutput.WriteString(string(headHtml))

		markdownInput, err := os.ReadFile(
			DRAFT_DIR + "/" + post + "/in.md")
		check(err)
		htmlOutput.WriteString(string(markdown.ToHTML(markdownInput, nil, nil)))

		footHtml, err := os.ReadFile(
			TEMPLATE_DIR + "/foot_post.html")
		check(err)
		htmlOutput.WriteString(string(footHtml))

		os.Rename(DRAFT_DIR+"/"+post, postDir+"/"+post)

		buf, err := os.OpenFile(
			postDir+"/"+post+"/in.md",
			os.O_RDONLY,
			0644)
		check(err)
		defer buf.Close()

		scanner := bufio.NewScanner(buf)
		// throw first line aka the title away
		scanner.Scan()
		scanner.Scan()
		postDate := scanner.Text()
		epochRegexp := regexp.MustCompile(`[0-9]{10}`)
		epochDate, _ := strconv.ParseInt(
			string(
				epochRegexp.Find(
					[]byte(postDate))), 10, 64)

		err = os.Chtimes(postDir+"/"+post,
			time.Unix(epochDate, 0),
			time.Unix(epochDate, 0))
		check(err)
	}

	backup_posts(postDir)
	buildIndex()
}

func editPost() {
	post := select_post("post to edit> ")
	splitPath := strings.Split(post, "/")
	postName := splitPath[len(splitPath)-1]

	err := os.Rename(postDir+"/"+postName, DRAFT_DIR+"/"+postName)

	if err != nil {
		panic(err)
	}

	fmt.Println("moved post " + postName + " to draft dir")
	os.RemoveAll(DRAFT_DIR + "/" + postName + "/index.html")

	launchEditor(EDITOR, DRAFT_DIR+"/"+postName+"/in.md")
}

func deletePost() {
	post := select_post("post to delete> ")
	splitPath := strings.Split(post, "/")
	postName := splitPath[len(splitPath)-1]

	err := os.RemoveAll(postDir + "/" + postName)

	if err != nil {
		panic(err)
	}

	fmt.Println("moved post " + postName + " to draft dir")
}

func backup_posts(dir string) {
	err := copy.Copy(dir, BACKUP_DIR)
	check(err)
}

func main() {
	cwd, _ := os.Getwd()

	for _, dir := range []string{BACKUP_DIR, DRAFT_DIR, postDir} {
		// throw away returned value if it's not an error
		// we simply want to check if the directory exists or not
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			os.MkdirAll(dir, 0755)
		}
	}

	if _, err := os.Stat(TEMPLATE_DIR); os.IsNotExist(err) {
		os.Rename(cwd+"/res/templates", WORKING_DIR+"/templates")
	}

	if len(os.Args) < 2 {
		fmt.Println("usage: sgb {")
		fmt.Printf("\tn(ew) [title]: create a new post\n")
		fmt.Printf("\te(dit): edit an existing post\n")
		fmt.Printf("\td(elete): delete an existing post\n")
		fmt.Printf("\tp(ublish): move drafts to post directory\n")
		fmt.Printf("\tf(etch): fetch posts from remote host\n")
		fmt.Printf("\ts(ync): use rsync to push local posts to remote web server\n")
		fmt.Println("}")
		os.Exit(1)
	}

	op := os.Args[1]

	switch op {
	case "n":
		if len(os.Args) > 2 {
			title := os.Args[2]
			createPost(title)
		} else {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("> ")
			title, err := reader.ReadString('\n')

			if err != nil {
				panic(err)
			}

			createPost(title)
		}
	case "e":
		editPost()
	case "d":
		deletePost()
	case "p":
		assemblePosts()
	case "f":
		fetchPosts()
	case "s":
		syncPosts()
	}

	buildIndex()
}
