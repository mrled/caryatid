# Caryatid

An [Atlas](https://atlas.hashicorp.com) is ["a support sculpted in the form of a man"](https://en.wikipedia.org/wiki/Atlas_(architecture)); a [Caryatid](https://github.com/mrled/packer-post-processor-caryatid) is such a support in the form of a [woman](https://en.wikipedia.org/wiki/Caryatid).

Caryatid is a minimal alternative to Atlas. It can build Vagrant catalogs and copy files to local or remote storage, and it can be invoked as a Packer post-processor.

More specifically, Caryatid intended as a way to host a (versioned) Vagrant catalog on systems without having to use (and pay for) Atlas, and without having to trust a third party if you don't want to. It's designed to work with dumb remote servers - all you need to have a remote scp catalog for example is a standard scp server, no server-side logic required. It supports multiple storage backends. For now, these are just scp and file backends; we would like to add support for more backends in the future (so long as they require no server-side logic).

Note that the file backend is useful even if the fileserver is local, because Vagrant needs the JSON catalog to be able to use versioned boxes. That is, using Caryatid to manage a JSON catalog of box versions is an improvement over running packer and then just doing a `vagrant box add` on the resulting box, because this way Vagrant can see when your box has a new version.

## Prerequisites

- Go
- Packer
- Disk space to keep (large) Vagrant box files

## Output and directory structure

Using a destination of `/srv/vagrant`, a box name of `testbox`, and trying to add a Virtualbox edition of that box at version 1.0.0 would result in a directory structure like this:

    /srv/vagrant
        /testbox.json: the JSON catalog
        /boxes
            /testbox_1.0.0_virtualbox.box: the large VM box file itself

And the `testbox.json` catalog will look like this:

    {
        "name": "testbox",
        "description": "a box for testing",
        "versions": [{
            "version": "1.0.0",
            "providers": [{
                "name": "virtualbox",
                "url": "file:///srv/vagrant/boxes/testbox_1.0.0.box",
                "checksum_type": "sha1",
                "checksum": "d3597dccfdc6953d0a6eff4a9e1903f44f72ab94"
            }]
        }]
    }

This can be consumed in a Vagrant file by using the JSON catalog as the box URL in a `Vagrantfile`:

    config.vm.box_url = "file:///srv/vagrant/testbox.json"

## Caveats

- Vagrant is [supposed to support scp](https://github.com/mitchellh/vagrant/pull/1041), but [apparently doesn't bundle a properly-built `curl` yet](https://github.com/mitchellh/vagrant-installers/issues/30). This means you may need to build your own `curl` that supports scp, and possibly even replace your system-supplied curl with that one, in order to use catalogs hosted on scp with Vagrant. (Note that we do not rely on curl, so even if your curl is old, Caryatid can still push to scp backends.)

## Roadmap / wishlist

- Would love to support S3 storage, however, there isn't a way to authenticate to S3 through Vagrant, at least without third party libraries. This would mean that the boxes stored on S3 would be public. This is fine for my use case, except that it means anyone with the URL to a box could cost me money just by downloading the boxes over and over
- Some sort of webserver mode would be nice, and is in line with the no server-side logic goal. Probably require an scp url for doing uploads in addition to an http url for vagrant to fetch the boxes? Or could require WebDAV?
- When is it appropriate/not appropriate to use `panic()` ?
- Instead of using the .box artifact filename to determine the provider, extract the metadata.json file from it and use that (see (the Vagrant docs on its .box fileformat)[https://www.vagrantup.com/docs/boxes/format.html]). This should be more robust

## See also

- The (Vagrant docs on the .box format)[https://www.vagrantup.com/docs/boxes/format.html] goes into detail on the "box metadata", that is, what I have been calling the "Vagrant catalog"
- [How to set up a self-hosted "vagrant cloud" with versioned, self-packaged vagrant boxes](https://github.com/hollodotme/Helpers/blob/master/Tutorials/vagrant/self-hosted-vagrant-boxes-with-versioning.md)
- [Distributing Vagrant base boxes securely](http://chase-seibert.github.io/blog/2014/05/18/vagrant-authenticated-private-box-urls.html)

## Packer artifact flow

In designing this plugin, it was helpful for me to outline how artifacts go from one step to another in a Packer workflow

- Build using one or more relevant builders.
    - This should be something that makes sense for Vagrant to consume, such as Virtualbox.
    - Note that each builder will result in a single *artifact* that is composed of one or more *files*. For instance, the VMware builder outputs at least one `.vmdk` disk file as well as a `.vmx` file describing the virtual machine, and perhaps others as well.
- Optionally run one or more provisioners, which do not result in different artifacts, but instead allow packer to modify the VMs by logging into them and applying configuration changes
- Run the Vagrant post-processor
    - This is necessary because Caryatid is, after all, intended for use with Vagrant
    - This will take an artifact potentially consisting of several files, do some Vagrant-related processing, and output an artifact consisting of just one file with a filename ending in `.box`.
    - Note that each post-processor gets run once per artifact, so if you have defined multiple builders (say, a Virtualbox builder and a VMware builder), you'll end up with two artifacts, and the Vagrant provisioner will run once per artifact.
    - Note that since Vagrant boxes are provider-specific, the Vagrant post-processor is hard-coded to understand various providers and those providers only. The supported providers are listed in the documentation: https://www.packer.io/docs/post-processors/vagrant.html
    - The output filename is by default `packer_{{.BuildName}}_{{.Provider}}.box`, where the `BuildName` and `Provider` variables are filled in automatically. The Caryatid post-processor relies on this being the default, so make sure not to override it
- Run the Caryatid post-processor
    - Just like Vagrant, if you have defined multiple builders, this post-processor will see one artifact per builder (and modified by the Vagrant post-processor)
    - Note that **we rely on the default Vagrant output filename**, because that's how we determine provider information
    - Based on configuration information specified in the packerfile, this will find a Vagrant catalog, add an entry for the relevant version and provider, and copy the Vagrant box file to a standard location linked from that catalog.
    - Note that by default it (and all Packer post-provisioners) will delete the input artifacts. This may be undesirable, for instance if your Vagrant catalog is on the Internet and you want to keep a local cache of it, so make sure to specify this in your packerfile.
