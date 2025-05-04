# Advanced Usage

DalleDress supports advanced workflows for power users and integrators.

## Custom Backends

You can implement your own backend to use a different AI model:

```go
type MyBackend struct{}

func (b *MyBackend) Generate(prompt string) (dalledress.Image, error) {
    // Custom model logic
}

ctx := dalledress.NewContextWithBackend(&MyBackend{})
```

## Batch Generation

Generate multiple images in a loop:

```go
for _, prompt := range prompts {
    img, err := ctx.GenerateImage(prompt)
    // ... handle img ...
}
```

## Image Annotation

Chain annotations and manipulations:

```go
img := ctx.GenerateImage("A cat in a hat")
img.Annotate("Dr. Seuss tribute").Resize(512, 512).Save("cat.png")
```

## Embedding in Applications

DalleDress can be embedded in CLI tools, web servers, or desktop apps. See the [Public Interface](04-public-interface.md) for core methods.
