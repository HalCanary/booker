```
Copyright 2022 Hal Canary Use of this program is governed by the file
LICENSE.

VARIABLES

var BookAlreadyExists = errors.New("Book Already Exists")
var UnsupportedUrlError = errors.New("unsupported url")
    Returned by a downloadFunction when the URL can not be handled.


FUNCTIONS

func CalculateLastModified(chapters []Chapter) time.Time
    Return the time of most recently modified chapter.

func EbookConvert(src, dst string, info EbookInfo) error
    Call Calibre's `ebook-convert` command with metadata from `info`.

func ExtractText(root *Node) string
    Extract and combine all Text Nodes under given node.

func GetAttribute(node *Node, key string) string
    Find the matching attributes, ignoring namespace.

func GetUrl(url, ref string, force bool) (io.ReadCloser, error)
    Fetch the content of a URL, using a cache if possible and if force is fakse.

func Humanize(v int) string
    Humanize converts a byte size to a human readable number, for example: 2048
    -> "2 kB"

func NormalizeString(v string) string
    Simplify and normalize a Unicode string.

func Register(downloadFunction func(url string) (EbookInfo, error))
    Register the given function.

func RenderDoc(w io.Writer, root *Node) error
    Generates HTML5 doc.

func SendFile(dst, path, contentType string, secrets EmailSecrets) error
    Send a file to a single destination.


TYPES

type Attachment struct {
	Filename    string
	ContentType string
	Data        []byte
}
    Attachment for an email.

type Chapter struct {
	Title    string
	Url      string
	Content  *Node
	Modified time.Time
}

type EbookInfo struct {
	Authors   string
	CoverURL  string
	CoverPath string
	Comments  string
	Title     string
	Source    string
	Language  string
	Chapters  []Chapter
	Modified  time.Time
}
    Ebook content and metadata.

func Download(url string) (EbookInfo, error)
    Return the result of the first registered download function that does not
    return UnsupportedUrlError.

func (info *EbookInfo) Write(directory string) (string, error)
    Write the ebook into given directory as HTML5 documents.}|

type Email struct {
	Date        time.Time
	To          []string
	Cc          []string
	Bcc         []string
	From        string
	Subject     string
	Content     string
	Attachments []Attachment
	Headers     map[string]string
}
    An electric mail message.

func (mail Email) Make() []byte
    Make, but do not send an email message.

func (m Email) Send(secrets EmailSecrets) error
    Send the given email using the provided SMTP secrets.

type EmailSecrets struct {
	SmtpHost string            `json:"SMTP_HOST"` // example: "smtp.gmail.com"
	SmtpUser string            `json:"SMTP_USER"` // example: "halcanary@gmail.com"
	SmtpPass string            `json:"SMTP_PASS"` // for gmail, is a App Password
	FromAddr string            `json:"FROM_ADDR"`
	Headers  map[string]string `json:"HEADERS"`
}
    Data structure representing instructions for connecting to SMTP server.
    Headers are additional headers to be added to outgoing email.

func GetSecrets(path string) (EmailSecrets, error)
    Read email secrets from the given file.

type Node = html.Node

func Cleanup(node *Node) *Node
    Clean up a HTML fragment.

func Comment(data string) *Node
    Return a HTML comment with the given data.

func Elem(tag string, children ...*Node) *Node
    Return an element with the given children.

func Element(tag string, attributes map[string]string, children ...*Node) *Node
    Return an element with given attributes and children.

func Remove(node *Node) *Node
    Remove a node from it's parent.

func TextNode(data string) *Node
    Return a HTML node with the given text.

```
