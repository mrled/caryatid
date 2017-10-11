# TODO

More specific / smaller items than the roadmap in the readme.

## On deck

 *  Is it a URL or a URI or what

    Like, get some consistency here, damn

 *  Improve test code, especially integration test code

    Integration tests tend to become multi-page monstrosities.
    Break these apart.

 *  Document when identical versions with different prerelease tags are considered to match and when they aren't.

    When querying like `=1.0.0`, we only return exact matches;
    we would not return e.g. `1.0.0-BETA` in this case.

    But when querying without a qualifier like `1.0.0`,
    or with a less-than/greater-than qualifier like `<1.0.0` or `<=1.0.0`,
    we *would* return e.g. `1.0.0-BETA`.

    This might be confusing.
    If we can't fix the confusion,
    we should at least document it.
    (It is already documented in comments;
    I mean documenting for the end user.)

    (Random thought: add `~`, `<~`, and `>~` for explicitly matching different prerelease tags?)

 *  Implement and test S3 backend

    I'm so ready

 *  Build a first class concept of a Vagrant Box into the Catalog

    Currently, there's a CaryatidOutputArtifact,
    which implements the *packer* concept of an *artifact*,
    but there is no *Caryatid* concept of a *Vagrant box*,
    particularly from the perspective of the Catalog.

    A box would have all the properties in a Catalog,
    and one would exist for each Catalog/Version/Provider combination.

    However, CatalogUri + VersionString + ProviderName would be all that is necessary to identify a box,
    so functions like `Equals()` should be written and documented carefully.
    
    (I have already dealt with this in a small way with BoxReference,
    but I'm thinking of something more comprehensive.)

 *  Improve CLI argument parsing and help

    1)  Subcommands

        The `docker` arguments are pretty complex, maybe go see what they're doing

    2)  Better messages to the user

        Subcommands are part of this,
        but basically make it more discoverable and less trial-and-error-y

    I strongly suspect this will require using something other than the standard `flags` module,
    which I have come to loathe.
    Maybe someone ported Python's `argparse`?

 *  Get a handle on logging

    1)  Support logging to a file

    2)  Maybe add debug logging? 
        There's lots of bullshit in the logs that end users will absolutely not care about

    3)  Definitely add verbose logging,
        controllable via a commandline flag

    4)  How should it be handled in the Packer plugin?
        Does Packer have a logging system it wants you to use?

## Far future

Not sure how feasible this stuff is, but it's on my mind

 *  Split out backends into completely separate projects.

    This would help in two ways:

    1)  If I change the CaryatidBackend interface,
        I will be able to update the individual backends more easily to match it,
        since I won't be facing the whole set of broken integration tests all at once

    2)  Other people can build their own without forking the repo or getting a PR accepted