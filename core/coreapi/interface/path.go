package iface

import (
	cid "gx/ipfs/QmYVNvtQkeZ6AKSwDrjQTs432QtL6umrrK41EBq3cu7iSP/go-cid"
)

// Path is a generic wrapper for paths used in the API. A path can be resolved
// to a CID using one of Resolve functions in the API.
//
// Paths must be prefixed with a valid prefix:
//
// * /ipfs - Immutable unixfs path (files)
// * /ipld - Immutable ipld path (data)
// * /ipns - Mutable names. Usually resolves to one of the immutable paths
//TODO: /local (MFS)
type Path interface {
	// String returns the path as a string.
	String() string

	// Namespace returns the first component of the path.
	//
	// For example path "/ipfs/QmHash", calling Namespace() will return "ipfs"
	Namespace() string

	// Mutable returns false if the data pointed to by this path in guaranteed
	// to not change.
	//
	// Note that resolved mutable path can be immutable.
	Mutable() bool
}

// ResolvedPath is a path which was resolved to the last resolvable node
type ResolvedPath interface {
	// Cid returns the CID of the node referenced by the path. Remainder of the
	// path is guaranteed to be within the node.
	//
	// Examples:
	// If you have 3 linked objects: QmRoot -> A -> B:
	//
	// cidB := {"foo": {"bar": 42 }}
	// cidA := {"B": {"/": cidB }}
	// cidRoot := {"A": {"/": cidA }}
	//
	// And resolve paths:
	// * "/ipfs/${cidRoot}"
	//   * Calling Cid() will return `cidRoot`
	//   * Calling Root() will return `cidRoot`
	//   * Calling Remainder() will return ``
	//
	// * "/ipfs/${cidRoot}/A"
	//   * Calling Cid() will return `cidA`
	//   * Calling Root() will return `cidRoot`
	//   * Calling Remainder() will return ``
	//
	// * "/ipfs/${cidRoot}/A/B/foo"
	//   * Calling Cid() will return `cidB`
	//   * Calling Root() will return `cidRoot`
	//   * Calling Remainder() will return `foo`
	//
	// * "/ipfs/${cidRoot}/A/B/foo/bar"
	//   * Calling Cid() will return `cidB`
	//   * Calling Root() will return `cidRoot`
	//   * Calling Remainder() will return `foo/bar`
	Cid() *cid.Cid

	// Root returns the CID of the root object of the path
	//
	// Example:
	// If you have 3 linked objects: QmRoot -> A -> B, and resolve path
	// "/ipfs/QmRoot/A/B", the Root method will return the CID of object QmRoot
	//
	// For more examples see the documentation of Cid() method
	Root() *cid.Cid

	// Remainder returns unresolved part of the path
	//
	// Example:
	// If you have 2 linked objects: QmRoot -> A, where A is a CBOR node
	// containing the following data:
	//
	// {"foo": {"bar": 42 }}
	//
	// When resolving "/ipld/QmRoot/A/foo/bar", Remainder will return "foo/bar"
	//
	// For more examples see the documentation of Cid() method
	Remainder() string

	Path
}
