# sgb - static go blog

## setup

compile the binary with flags of your choice

```
go build -o ./bin/sgb -ldflags="-s -w" ./src
```

then, symlink or copy the binary to a directory present in your $PATH, e.g. `/opt`

## usage

```
$ sgb
usage: sgb {
        n(ew) [title]: create a new post
        e(dit): edit an existing post
        d(elete): delete an existing post
        p(ublish): move drafts to post directory
        f(etch): fetch posts from remote host
        s(ync): use rsync to push local posts to remote web server
}
```

### options

+ `n` creates a new directory with a stripped title of your choice containing a markdown file `in.md`. the title is either prompted for or passed as an argument to `n`
+ `p` publishes your drafts to the post directory `out/p/`
+ `e` moves an assembled post from the post directory back to the drafts directory `drafts/` and opens vim
+ `d` removes an assembled post from the post directory
+ `f` fetches post files from your remote web server and places them in the draft directory, erasing the generated post html files and the remote directory in the process
+ `s` syncs local published posts to your remote web server

### directory structure

sgb will create the following directory structure:

```
$HOME/Documents
└ blog/
  ├ bak/
  ├ drafts/
  ├ templates/
  └ out/
    └ p/
```

+ `bak/` will contain the raw markdown drafts that get pushed to the `out/p` directory using the `p` operation
+ `drafts/` will contain new posts that get created using the `n` operation
+ `templates/` will contain html and xml header and footer files used to assemble the static files
+ `out/` will contain `index.html` which will list your existing blog posts
+ `out/p/` will contain your assembled post files in markdown and html format
