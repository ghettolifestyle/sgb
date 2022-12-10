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
)

var EDITOR string = "/usr/bin/vi"
var WORKING_DIR string = "/Users/m/Documents/blog"
var BACKUP_DIR string = "/Users/m/Documents/blog-bak"
var DRAFT_DIR string = WORKING_DIR + "/drafts"
var TEMPLATE_DIR string = WORKING_DIR + "/templates"
var OUT_DIR string = WORKING_DIR + "/out"
var POST_DIR string = OUT_DIR + "/p"
var REMOTE_ROOT_DIR string = "/var/static"

var SSH_USER string = "root"
var SSH_HOST string = "kmai.xyz"
var SSH_PORT int = 22

func check(err error) {
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
}

func sync_posts() {
	rsync_arguments := OUT_DIR + "/* " + SSH_USER + "@" + SSH_HOST + ":" + REMOTE_ROOT_DIR + "/"

	cmd := exec.Command("bash", "-c", "rsync -e \"ssh -i $HOME/.ssh/id_ed25519\" -a "+rsync_arguments)
	err := cmd.Run()
	check(err)
}

func fetch_posts() {
	posts, _ := os.ReadDir(POST_DIR)

	for _, post := range posts {
		os.RemoveAll(POST_DIR + "/" + post.Name())
	}

	rsync_arguments := SSH_USER + "@" + SSH_HOST + ":" + REMOTE_ROOT_DIR + "/p/ " + DRAFT_DIR + "/"

	cmd := exec.Command("bash", "-c", "rsync -e \"ssh -i $HOME/.ssh/id_ed25519\" -a "+rsync_arguments)
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

func launch_editor(editor string, file string) {
	cmd := exec.Command(editor, file)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout

	err := cmd.Run()
	check(err)
}

func select_post(prompt string) string {
	posts, err := os.ReadDir(POST_DIR)
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

	return POST_DIR + "/" + posts[selection].Name()
}

func create_post(title string) {
	nonalphanumeric_regexp := regexp.MustCompile(`\W`)
	multi_underscore_regexp := regexp.MustCompile(`[_]{2,}`)

	title = strings.ReplaceAll(title, "\n", "")
	date := strconv.FormatInt(time.Now().Unix(), 10)
	buf := []byte(strings.ReplaceAll(title, " ", "_"))
	formatted_title := nonalphanumeric_regexp.ReplaceAll(buf, []byte(""))
	formatted_title = multi_underscore_regexp.ReplaceAll(formatted_title, []byte("_"))
	fmt.Println(string(formatted_title))

	post_dir := DRAFT_DIR + "/" + string(formatted_title)
	post_file := post_dir + "/in.md"

	os.Mkdir(post_dir, 0755)
	file, err := os.OpenFile(post_file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	check(err)

	defer file.Close()

	file.WriteString("# " + title + "\n")
	file.WriteString("<span class=\"date\">" + date + "</span>\n")
	file.WriteString("\n")

	fmt.Println("created post " + title + " in draft dir")

	launch_editor(EDITOR, post_file)
}

func build_index() {
	// remove old index file
	os.RemoveAll(OUT_DIR + "/index.html")

	posts, err := os.ReadDir(POST_DIR)
	check(err)

	sort.Slice(posts, func(i, j int) bool {
		file_info_i, _ := posts[i].Info()
		file_info_j, _ := posts[j].Info()
		return file_info_i.ModTime().After(file_info_j.ModTime())
	})

	index_file, err := os.OpenFile(
		OUT_DIR+"/index.html",
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644)
	check(err)
	defer index_file.Close()

	head_html, err := os.ReadFile(TEMPLATE_DIR + "/head_index.html")
	check(err)
	index_file.Write(head_html)

	index_file.WriteString("<p>some words on pages about stuff</p>\n")
	index_file.WriteString("<h2>posts</h2>\n")
	index_file.WriteString("<ul id=\"posts\">\n")

	for _, post := range posts {
		post_name := post.Name()
		post_file, err := os.OpenFile(
			POST_DIR+"/"+post_name+"/in.md",
			os.O_RDONLY,
			0644)
		check(err)

		// read first line of markdown input containing the post title
		// remove leading hashtag using slice
		scanner := bufio.NewScanner(post_file)
		scanner.Scan()
		post_title := scanner.Text()[2:]

		// read second line of markdown containing post date
		// in epoch format wrapped by span, e.g. <span class="date">1234567890</span>
		// use compiled regex to remove the span tags
		scanner.Scan()
		post_date := scanner.Text()
		date_regex := regexp.MustCompile(`[0-9]{10}`)
		// convert regex result to byte array to pass to Find(), then convert back to string
		regex_result := string(date_regex.Find([]byte(post_date)))
		// convert epoch date string to int64 in decimal
		// goal is to pass this int64 to time.Unix() to get proper Time type
		regex_result_int64, err := strconv.ParseInt(regex_result, 10, 64)
		check(err)

		formatted_date := strings.ToLower(time.Unix(regex_result_int64, 0).Format("Jan 02, 2006"))

		// set atime, mtime to epoch timestamp first generated when post was created
		// allows sorting when index file is built and post list is generated

		post_file.Close()

		index_file.WriteString("<li><a href=\"p/" + post_name + "\">" + post_title + "</a>\n")
		index_file.WriteString("<span class=\"date\">" + formatted_date + "</span></li>\n")
	}

	index_file.WriteString("</ul>\n")

	foot_html, err := os.ReadFile(TEMPLATE_DIR + "/foot_index.html")
	check(err)
	index_file.Write(foot_html)
}

func assemble_posts() {
	posts, err := os.ReadDir(DRAFT_DIR)
	check(err)

	for _, post := range posts {
		post := post.Name()

		html_output, err := os.OpenFile(
			DRAFT_DIR+"/"+post+"/index.html",
			os.O_APPEND|os.O_CREATE|os.O_WRONLY,
			0644)
		check(err)
		defer html_output.Close()

		head_html, err := os.ReadFile(
			TEMPLATE_DIR + "/head_post.html")
		check(err)
		html_output.WriteString(string(head_html))

		markdown_input, err := os.ReadFile(
			DRAFT_DIR + "/" + post + "/in.md")
		check(err)
		html_output.WriteString(string(markdown.ToHTML(markdown_input, nil, nil)))

		foot_html, err := os.ReadFile(
			TEMPLATE_DIR + "/foot_post.html")
		check(err)
		html_output.WriteString(string(foot_html))

		os.Rename(DRAFT_DIR+"/"+post, POST_DIR+"/"+post)

		buf, err := os.OpenFile(
			POST_DIR+"/"+post+"/in.md",
			os.O_RDONLY,
			0644)
		check(err)
		defer buf.Close()

		scanner := bufio.NewScanner(buf)
		// throw first line aka the title away
		scanner.Scan()
		scanner.Scan()
		post_date := scanner.Text()
		epoch_regexp := regexp.MustCompile(`[0-9]{10}`)
		epoch_date, _ := strconv.ParseInt(
			string(
				epoch_regexp.Find(
					[]byte(post_date))), 10, 64)

		err = os.Chtimes(POST_DIR+"/"+post,
			time.Unix(epoch_date, 0),
			time.Unix(epoch_date, 0))
		check(err)
	}

	build_index()
}

func edit_post() {
	post := select_post("post to edit> ")
	split_path := strings.Split(post, "/")
	post_name := split_path[len(split_path)-1]

	err := os.Rename(POST_DIR+"/"+post_name, DRAFT_DIR+"/"+post_name)

	if err != nil {
		panic(err)
	}

	fmt.Println("moved post " + post_name + " to draft dir")
	os.RemoveAll(DRAFT_DIR + "/" + post_name + "/index.html")
}

func delete_post() {
	post := select_post("post to delete> ")
	split_path := strings.Split(post, "/")
	post_name := split_path[len(split_path)-1]

	err := os.RemoveAll(POST_DIR + "/" + post_name)

	if err != nil {
		panic(err)
	}

	fmt.Println("moved post " + post_name + " to draft dir")
}

func main() {
	cwd, _ := os.Getwd()

	for _, dir := range []string{BACKUP_DIR, DRAFT_DIR, POST_DIR} {
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
		fmt.Println("}")
		os.Exit(1)
	}

	op := os.Args[1]

	switch op {
	case "n":
		if len(os.Args) > 2 {
			title := os.Args[2]
			create_post(title)
		} else {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("> ")
			title, err := reader.ReadString('\n')

			if err != nil {
				panic(err)
			}

			create_post(title)
		}
	case "e":
		edit_post()
	case "d":
		delete_post()
	case "p":
		assemble_posts()
	case "f":
		fetch_posts()
	case "s":
		sync_posts()
	}

	build_index()
}
