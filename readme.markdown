# Caryatid

An [Atlas](https://atlas.hashicorp.com) is "[a support sculpted in the form of a man](https://en.wikipedia.org/wiki/Atlas_(architecture))"; a [Caryatid](https://github.com/mrled/caryatid) is [such a support in the form of a woman](https://en.wikipedia.org/wiki/Caryatid).

Caryatid is a packer post-processor that can generate or update Vagrant catalogs on local storage. Vagrant will read versioning information from these catalogs and detect when there is a new version of the box available, which is not possible when just doing `vagrant box add`.

In the future, it will support remote catalogs, like scp, as well.

## Prerequisites

- Go
- Packer
- Disk space to keep (large) Vagrant box files

Caryatid is intended to work on any platform that Packer supports, but gets somewhat less testing on Windows. If you find something that's broken, please open an issue.

## Building and installing

 -  Build the binaries by changing to `./cmd/<projectname>` and running `go build`

    Building this way will result in a binary being built in the same directory

 -  Install the binaries by copying them to the right location

    A command-line tool such as `cmd/caryatid` might be copied to somewhere in `$PATH`

    A packer plugin such as `cmd/packer-post-processor-caryatid` must be copied to `~/.packer.d/plugins` or `%APPDATA%\packer.d\plugins`; see also the [official plugins documentation](https://www.packer.io/docs/extend/plugins.html)

 -  Build all projects with `go generate ./... && go build ./...` in the root directory

    Note that this immediately throws away the resulting binaries, and is intended just for testing that the build succeeds

 -  Test all projects with `go generate ./... && go test ./...` in the root directory

 -  Build all architectures for a release with `go run scripts/buildrelease.go`

    This will result in a `release` directory that contains executables for every supported architecture

## Using the Packer plugin

In your packerfile, you must add it as a post-processor in a *series*, and coming after a vagrant post-processor (because caryatid requires Vagrant boxes to come in as artifacts).

There are five configuration parameters:

- `name` (required): The name of the box.
- `description` (required): A longer description for the box
- `version` (required): The version of the box
    - Sometimes, it makes sense to set this based on the date; setting the version to `"1.0.{{isotime \"20060102150405\"}}"` will result in a version number of 1.0.YYYYMMDDhhmmss
    - This can be especially useful during development, so that you don't have to pass an ever-incrementing version number variable to `packer build`
    - See the `isotime` global function in the [packer documentation for configuration templates](https://www.packer.io/docs/templates/configuration-templates.html) for more information
- `catalog_url` (required): A URL for the directory containing the catalog
    - Note that Caryatid assumes the catalog name is always just `<box name>.json`
    - See the "Output and directory structure" section for more information
    - Interpreted individually by each backend
- `keep_input_artifact` (optional): Keep a copy of the Vagrant box at whatever location the Vagrant post-processor stored its output
    - By default, input artifacts are deleted; this suppresses that behavior, and will result in two copies of the Vagrant box on your filesystem - one where the Vagrant post-processor was configured to store its output, and one where Caryatid will copy it
- `backend`: The name of the backend to use. Currently only `file` and `s3` are supported

That might look like this:

    "variables": {
      "boxname": "wintriallab-win10-32",
      "version": "1.0.{{isotime \"20060102150405\"}}",
      "description": "Windows Trial Lab: Windows 10 x86",
      "catalog_url": "file://{{env `HOME`}}/wintriallab-win10-32.json"
    },
    ...<snip>...
    "post-processors": [
      [
        { "type": "vagrant", },
        {
          "type": "caryatid",
          "name": "{{user `boxname`}}",
          "version": "{{user `version`}}",
          "description": "{{user `description`}}",
          "catalog_url": "{{user `catalog_url`}}"
        }
      ]
    ]

### Note: post-processor series

See the double open square brackets (`[`) after `"post-processors":`? The first square bracket indicates the start of the `post-processors` section; the second indicates the start of a post-processor sequence, where artifacts from the previous post-processor are fed as input into the next.

If you don't define a sequence using that extra set of square brackets, but instead just place the vagrant and caryatid entries in the `post-processors` section directly, the vagrant post-processor will run with inputs from the builder, and then caryatid post-processor will run afterwards also with inputs from the builder, rather than with inputs from the vagrant post-processor.

See the [official post-processor documentation](https://www.packer.io/docs/templates/post-processors.html) for more details on sequences.

## Backends

 -  LocalFile:
     -  Requires URIs like `file:///path/to/somewhere` on Unix, or `file:///C:\\path\\to\\somewhere` on Windows
     -  Files created with the LocalFile backend conform to OS default permissions. On Unix, this means it honors `umask`; on Windows, this means it inherits directory permissions. When modifying a file, such as adding a box to an existing catalog, permissions of the existing file are not changed.
 -  S3:
     -  Requires URIs like `s3://bucket/key`, where `key` may include a directory name, e.g. in `s3://bucket/some/sub/path`, `some/sub/path` is the `key`. These URIs are supported by the [vagrant-s3auth](https://github.com/WhoopInc/vagrant-s3auth) plugin and may be used in Vagrant files where that plugin is installed
     -  Note that HTTP URIs like `http://s3.amazonaws.com/bucket/resource` are not supported, even though they are supported by the [vagrant-s3auth](https://github.com/WhoopInc/vagrant-s3auth) plugin.
     -  S3 permissions are not modified
     -  Requires credentials and a default region set in `~/.aws/credentials` and `~/.aws/config` respectively. The easiest way to do this is to [install the AWS CLI](http://docs.aws.amazon.com/cli/latest/userguide/cli-chap-getting-started.html) and run `aws configure`, making sure to provide responses for `AWS Access Key ID`, `AWS Secret Access Key`, and `Default region name` when prompted.

## Output and directory structure

Using a destination of `/srv/vagrant`, a box name of `testbox`, and trying to add a Virtualbox edition of that box at version 1.0.0 would result in a directory structure like this:

    /srv/vagrant
        /testbox.json: the JSON catalog
        /testbox
            /testbox_1.0.0_virtualbox.box: the large VM box file itself

And the `testbox.json` catalog will look like this:

    {
        "name": "testbox",
        "description": "a box for testing",
        "versions": [{
            "version": "1.0.0",
            "providers": [{
                "name": "virtualbox",
                "url": "file:///srv/vagrant/testbox/testbox_1.0.0.box",
                "checksum_type": "sha1",
                "checksum": "d3597dccfdc6953d0a6eff4a9e1903f44f72ab94"
            }]
        }]
    }

This can be consumed in a Vagrant file by using the JSON catalog as the box URL in a `Vagrantfile`:

    config.vm.box_url = "file:///srv/vagrant/testbox.json"

## Roadmap / wishlist

### SCP backend

Vagrant is [supposed to support scp](https://github.com/mitchellh/vagrant/pull/1041), but [apparently doesn't bundle a properly-built `curl` yet](https://github.com/mitchellh/vagrant-installers/issues/30). This means you may need to build your own `curl` that supports scp, and possibly even replace your system-supplied curl with that one, in order to use catalogs hosted on scp with Vagrant. (Note that Caryatid will not rely on curl, so even if your curl is old, we will still be able to push to scp backends; the only concern is whether your system's Vagrant can pull from them by default or not.)

### Webserver backend

WebDAV is a possibility, but I'm not sure whether it would be truly valuable or not - I don't see a lot of WebDAV servers out in the wild.

### Command line manager tool

Write a command line tool that can be used to inspect and modify the catalog.

- Query the catalog to determine what versions of a box are available and for which providers
- Delete box files by version and provider
- Delete all versions of a box older than some specified version
- Unclear what happens to Vagrant if you have a certain version of a box locally, but since that version was downloaded, subsequent versions were added to the catalog and the version that's still local is deleted. If this causes problems for Vagrant, perhaps instead of deleting the box file, I'd keep it in the backend's storage system and overwrite the large box file with an empty file?

Particularly for S3 storage, this will be useful to not only know what is available, but also to save money by deleting ancient unused versions of boxes.

If we add an HTTP backend, the tool would also be useful for other Vagrant catalogs that are not managed by Caryatid.

### Separate backend and frontend URIs

For a webserver backend, we could provide a backend SCP or LocalFile URI, and a frontend HTTP URI. For an S3 backend where boxes can be public, we could provide a backend S3 and a frontend HTTP URI (note that in the S3/HTTP case, the boxes must be public, since S3 doesn't support HTTP basic auth). In these cases, we wouldn't need to support WebDAV (which provides create/update/delete access), only vanilla HTTP/HTTPS (which of course can provide read access), because we could use backends we already ahve like LocalFile that provide C/U/D access.

This would let us keep with our goal of no serverside logic, while allowing for more frontends.
