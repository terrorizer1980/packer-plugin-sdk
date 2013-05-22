package rpc

import (
	"github.com/mitchellh/packer/packer"
	"net/rpc"
)

// An implementation of packer.Build where the build is actually executed
// over an RPC connection.
type build struct {
	client *rpc.Client
}

// BuildServer wraps a packer.Build implementation and makes it exportable
// as part of a Golang RPC server.
type BuildServer struct {
	build packer.Build
}

type BuildPrepareArgs interface{}

type BuildRunArgs struct {
	UiRPCAddress string
}

func Build(client *rpc.Client) *build {
	return &build{client}
}

func (b *build) Name() (result string) {
	b.client.Call("Build.Name", new(interface{}), &result)
	return
}

func (b *build) Prepare() (err error) {
	b.client.Call("Build.Prepare", new(interface{}), &err)
	return
}

func (b *build) Run(ui packer.Ui) packer.Artifact {
	// Create and start the server for the UI
	// TODO: Error handling
	server := rpc.NewServer()
	RegisterUi(server, ui)
	args := &BuildRunArgs{serveSingleConn(server)}

	var reply string
	if err := b.client.Call("Build.Run", args, &reply); err != nil {
		panic(err)
	}

	client, err := rpc.Dial("tcp", reply)
	if err != nil {
		panic(err)
	}

	return Artifact(client)
}

func (b *BuildServer) Name(args *interface{}, reply *string) error {
	*reply = b.build.Name()
	return nil
}

func (b *BuildServer) Prepare(args *BuildPrepareArgs, reply *error) error {
	*reply = b.build.Prepare()
	return nil
}

func (b *BuildServer) Run(args *BuildRunArgs, reply *string) error {
	client, err := rpc.Dial("tcp", args.UiRPCAddress)
	if err != nil {
		return err
	}

	artifact := b.build.Run(&Ui{client})

	// Wrap the artifact
	server := rpc.NewServer()
	RegisterArtifact(server, artifact)

	*reply = serveSingleConn(server)
	return nil
}