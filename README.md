macaroon command
-------

This command implements command-line support for creating, discharging
and checking macaroons.

A "macaroons" argument to a command specifies a list of unbound macaroons,
with the first element being the primary, or root, macaroon and the rest
being discharges. A macaroons argument can be specified in one of the
following ways:

 - a JSON string containing a object in bakery.Macaroon or macaroon.Macaroon format.
 - a JSON string containing an array of macaroons in bakery.Macaroon format.
 - a base64-encoded string containing any of the above.
 - any of the above prefixed with "unbound:".

A macaroon list is printed as a base64-encoded
string holding a JSON array holding macaroons in bakery.Macaroon
format, prefixed with the string "unbound:".

The base64 encoding will not include terminating "=" padding
characters.

A format argument can be one of the following:
	json			- base64 JSON encoded
	binary		- base64 binary encoded
	rawjson		- JSON encoded
	rawbinary	- binary encoded
Binary formats can only be used with bound macaroons.

	macaroon new [--expiry duration] op...

Create new macaroon valid for the given operations,
which expires after the given duration from now.

	macaroon check [--any] op... [macaroons...]

Return status indicating whether macaroons are allowed to perform all given
operations. If --any flag is given, print allowed status of all operations.
All the macaroons should be bound (for example using the use command).
e.g. macaroon check read:/usr/bin/x dfvnmdsvflkfdjsnvldksnv dsakhjcsdhjcbsk

	macaroon discharge macaroon

Acquire any discharges needed for the undischarged macaroon
and print macaroon slice.

	macaroon use [--format format] macaroons

Use macaroons in a request. Takes the given macaroons, which
must have include all discharges, and prints it in the specified
format (default binary).

	macaroon caveat [-3 location] [--public-key xxxx] macaroon condition

Add caveat to macaroon, prints new macaroon.
looks up public key of location if not provided
(could use local cache)

	macaroon show [--format text|json|binary] macaroons

Show macaroons formatted with the given
format. If --raw is specified, binary output will not be base64-quoted.


UNIMPLEMENTED AS YET

	macaroon newkey
	
Generate a new public-private key pair and print it.

	macaroon login
	
Log into the local macaroon root key server. Prints:

	export ROOTKEY_MACAROON=xxxxx

All commands recognize that env var and use it
to talk to the server.

	macaroon discharger json-spec
	
Run 3rd party caveat discharge service
json spec maps caveat conditions to required operations.
you can discharge a caveat condition if the caveat condition
matches a pattern and the provided discharge token contains
a set of macaroons that allows the associated operations.
e.g.

	{
		condition: "answered-quiz ([a-z]+)",
		ops: ["answered:\1"],
	}

Encrypt macaroons at rest; use a server listening on a unix socket
to retrieve and create root keys. To obtain access to the server,
authenticate somehow (initially just a password, later perhaps user
too), which gives us a macaroon that will allow access to the rest
of the API.

Initially, just use a single root key, encrypted with the password.
Send password hashed with password-strong hash.
