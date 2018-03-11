package options

type ObjectNewSettings struct {
	Type string
}

type ObjectPutSettings struct {
	InputEnc string
	DataType string
}

type ObjectAddLinkSettings struct {
	Create bool
}

type ObjectNewOption func(*ObjectNewSettings) error
type ObjectPutOption func(*ObjectPutSettings) error
type ObjectAddLinkOption func(*ObjectAddLinkSettings) error

func ObjectNewOptions(opts ...ObjectNewOption) (*ObjectNewSettings, error) {
	options := &ObjectNewSettings{
		Type: "empty",
	}

	for _, opt := range opts {
		err := opt(options)
		if err != nil {
			return nil, err
		}
	}
	return options, nil
}

func ObjectPutOptions(opts ...ObjectPutOption) (*ObjectPutSettings, error) {
	options := &ObjectPutSettings{
		InputEnc: "json",
		DataType: "text",
	}

	for _, opt := range opts {
		err := opt(options)
		if err != nil {
			return nil, err
		}
	}
	return options, nil
}

func ObjectAddLinkOptions(opts ...ObjectAddLinkOption) (*ObjectAddLinkSettings, error) {
	options := &ObjectAddLinkSettings{
		Create: false,
	}

	for _, opt := range opts {
		err := opt(options)
		if err != nil {
			return nil, err
		}
	}
	return options, nil
}

type objectOpts struct{}

var Object objectOpts

// Type is an option for Object.New which allows to change the type of created
// dag node.
//
// Supported types:
// * 'empty' - Empty node
// * 'unixfs-dir' - Empty UnixFS directory
func (_ objectOpts) Type(t string) ObjectNewOption {
	return func(settings *ObjectNewSettings) error {
		settings.Type = t
		return nil
	}
}

// InputEnc is an option for Object.Put which specifies the input encoding of the
// data. Default is "json".
//
// Supported encodings:
// * "protobuf"
// * "json"
func (_ objectOpts) InputEnc(e string) ObjectPutOption {
	return func(settings *ObjectPutSettings) error {
		settings.InputEnc = e
		return nil
	}
}

// DataType is an option for Object.Put which specifies the encoding of data
// field when using Json or XML input encoding.
//
// Supported types:
// * "text" (default)
// * "base64"
func (_ objectOpts) DataType(t string) ObjectPutOption {
	return func(settings *ObjectPutSettings) error {
		settings.DataType = t
		return nil
	}
}

// Create is an option for Object.AddLink which specifies whether create required
// directories for the child
func (_ objectOpts) Create(create bool) ObjectAddLinkOption {
	return func(settings *ObjectAddLinkSettings) error {
		settings.Create = create
		return nil
	}
}
