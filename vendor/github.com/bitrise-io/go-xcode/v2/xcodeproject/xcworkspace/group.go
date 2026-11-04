package xcworkspace

// Group is a folder in the workspace's file tree. Its members' group-relative locations resolve
// against it.
type Group struct {
	Location string    `xml:"location,attr"`
	FileRefs []FileRef `xml:"FileRef"`
	Groups   []Group   `xml:"Group"`
}
