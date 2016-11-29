/*
In architecture, an Atlas [https://atlas.hashicorp.com/] is a column or support sculpted in the form of a man; a Caryatid [https://github.com/mrled/packer-post-processor-caryatid] is such a support in the form of a woman.

Caryatid is a packer post-processor plugin that provides a way to host a (versioned) Vagrant catalog on systems without having to use (and pay for) Atlas, and without having to trust a third party unless you want to.

Here's the JSON of an example catalog:

	{
		"name": "testbox",
		"description": "Just an example",
		"versions": [{
			"version": "0.1.0",
			"providers": [{
				"name": "virtualbox",
				"url": "user@example.com/caryatid/boxes/testbox_0.1.0.box",
				"checksum_type": "sha1",
				"checksum": "d3597dccfdc6953d0a6eff4a9e1903f44f72ab94"
			}]
		}]
	}
*/
package caryatid

type Provider struct {
	Name         string `json:"name"`
	Url          string `json:"url"`
	ChecksumType string `json:"checksum_type"`
	Checksum     string `json:"checksum"`
}
type Version struct {
	Version   string     `json:"version"`
	Providers []Provider `json:"providers"`
}
type Catalog struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Versions    []Version `json:"versions"`
}
