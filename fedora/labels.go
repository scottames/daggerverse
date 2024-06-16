package main

// ContainerLabel represents a Label to be placed in the generated Container
// image at the Destination
type ContainerLabel struct {
	Name  string
	Value string
}

// WithLabel will append a label to the generated Container image
//
// maps not currently supported: https://github.com/dagger/dagger/issues/6138
// if they do become supported this should be simplified to allow passing a
// map
func (f *Fedora) WithLabel(name string, value string) *Fedora {
	f.Labels = append(f.Labels, &ContainerLabel{
		Name:  name,
		Value: value,
	})

	return f
}

// WithDescription will append a label to the generated Container image
// with the given description
//
//	example: org.opencontainers.image.description=<description>
func (f *Fedora) WithDescription(
	// description to be added to the generated Container image
	description string,
) *Fedora {
	f.Labels = append(f.Labels, &ContainerLabel{
		Name:  "org.opencontainers.image.description",
		Value: description,
	})

	return f
}
