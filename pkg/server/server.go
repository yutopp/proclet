package server

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"connectrpc.com/connect"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/pkg/errors"
	apiv1pb "github.com/yutopp/koya/pkg/proto/api/v1"
	v1 "github.com/yutopp/koya/pkg/proto/api/v1"
	apiv1connect "github.com/yutopp/koya/pkg/proto/api/v1/v1connect"
	"github.com/yutopp/koya/pkg/service/container"
)

type Config struct {
	TempDir   string
	RunnerUID int
	RunnerGID int

	Logger *zap.Logger
}

// Server implements the KoyaServiceServer interface
type Server struct {
	config *Config
}

var _ apiv1connect.KoyaServiceHandler = (*Server)(nil)

func Register(mux *http.ServeMux, srv *Server) {
	loggingInterceptor := NewLoggingInterceptor(srv.config.Logger)
	path, handler := apiv1connect.NewKoyaServiceHandler(srv, connect.WithInterceptors(loggingInterceptor))
	mux.Handle(path, handler)
}

func NewServer(c *Config) *Server {
	return &Server{
		config: c,
	}
}

func (s *Server) List(context.Context, *connect.Request[emptypb.Empty]) (*connect.Response[v1.ListResponse], error) {
	res := &v1.ListResponse{
		Languages: make([]*apiv1pb.Language, 0, len(L)),
	}
	for _, l := range L {
		lang := &apiv1pb.Language{
			Id:       l.ID,
			ShowName: l.ShowName,
		}
		for _, p := range l.Processors {
			proc := &apiv1pb.Processor{
				Id:       p.ID,
				ShowName: p.ShowName,

				DefaultFilename: p.DefaultFilename,
			}
			for _, t := range p.Tasks {
				task := &apiv1pb.Task{
					Id:       t.ID,
					ShowName: t.ShowName,

					Kind: t.Kind,
				}
				if t.Compile != nil {
					task.Compile = &apiv1pb.PhasedTask{
						// Cmd: t.Compile.Cmd,
					}
				}
				if t.Run != nil {
					task.Run = &apiv1pb.PhasedTask{
						// Cmd: t.Run.Cmd,
					}
				}

				proc.Tasks = append(proc.Tasks, task)
			}

			lang.Processors = append(lang.Processors, proc)
		}

		res.Languages = append(res.Languages, lang)
	}

	return &connect.Response[v1.ListResponse]{
		Msg: res,
	}, nil
}

func (s *Server) RunOneshot(
	ctx context.Context,
	req *connect.Request[apiv1pb.RunOneshotRequest],
	stream *connect.ServerStream[apiv1pb.RunOneshotResponse],
) error {
	_, proc, task, err := lookupLanguage(req.Msg.LanguageId, req.Msg.ProcessorId, req.Msg.TaskId)
	if err != nil {
		return err
	}

	dirName, err := os.MkdirTemp(s.config.TempDir, "proclet-")
	if err != nil {
		return err
	}
	if err := os.Chown(dirName, s.config.RunnerUID, s.config.RunnerGID); err != nil {
		return err
	}
	log.Printf("directory created: %s", dirName)

	for _, file := range req.Msg.Files {
		path := filepath.Join(dirName, filepath.Clean(filepath.Join("/", file.Path)))
		if err := os.WriteFile(path, file.Content, 0755); err != nil {
			return err
		}
		if err := os.Chown(path, s.config.RunnerUID, s.config.RunnerGID); err != nil {
			return err
		}
	}

	c := &executeConfig{
		Image:    proc.DockerImage,
		ShellCmd: "",

		RunnerUID: s.config.RunnerUID,
		RunnerGID: s.config.RunnerGID,
		DirName:   dirName,

		Stream: stream,
	}
	if task.Compile != nil {
		if err := executePhase(ctx, c, task.Compile); err != nil {
			return err
		}
	}

	if task.Run != nil {
		if err := executePhase(ctx, c, task.Run); err != nil {
			return err
		}
	}

	log.Println("rpc finished")

	return nil
}

type executeConfig struct {
	Image    string
	ShellCmd string

	RunnerUID int
	RunnerGID int
	DirName   string

	Stream *connect.ServerStream[apiv1pb.RunOneshotResponse]
}

func buildShellCmd(cmd []string) string {
	// TODO: escape
	return strings.Join(cmd, " ")
}

func executePhase(ctx context.Context, c *executeConfig, phase *PhasedTask) error {
	e := container.NewDockerRunner()

	stdoutR, stdoutW := io.Pipe()
	stderrR, stderrW := io.Pipe()
	resourceLimit := container.ResourceLimits{
		Core:    0,                // Process can NOT create CORE file
		Nofile:  512,              // Process can open 512 files
		NProc:   30,               // Process can create processes to 30
		MemLock: 1024,             // Process can lock 1024 Bytes by mlock(2)
		CPUTime: 5,                // sec
		Memory:  10 * 1024 * 1024, // bytes
		FSize:   5 * 1024 * 1024,  // Process can writes a file only 5MiB
	}
	containerTask := &container.RunTask{
		Image:    c.Image,
		ShellCmd: buildShellCmd(phase.Cmd),

		UID:         c.RunnerUID,
		GID:         c.RunnerGID,
		HomeHostDir: c.DirName,

		Stdin:  nil,
		Stdout: stdoutW,
		Stderr: stderrW,

		Limits: resourceLimit,
	}
	handle, err := e.Run(ctx, containerTask)
	if err != nil {
		return err
	}

	var mu sync.Mutex // Stream is not thread-safe...
	var ioWg sync.WaitGroup
	ioWg.Add(2) // stdout, stderr
	go redirect(ctx, &ioWg, stdoutR, func(buf []byte) error {
		mu.Lock()
		defer mu.Unlock()

		outVal := &apiv1pb.Output{
			Kind:   0, // stdout
			Buffer: buf,
		}
		if err := c.Stream.Send(&apiv1pb.RunOneshotResponse{Response: &apiv1pb.RunOneshotResponse_Output{Output: outVal}}); err != nil {
			return err
		}

		return nil
	})
	go redirect(ctx, &ioWg, stderrR, func(buf []byte) error {
		mu.Lock()
		defer mu.Unlock()

		outVal := &apiv1pb.Output{
			Kind:   1, // stderr
			Buffer: buf,
		}
		if err := c.Stream.Send(&apiv1pb.RunOneshotResponse{Response: &apiv1pb.RunOneshotResponse_Output{Output: outVal}}); err != nil {
			return err
		}

		return nil
	})
	ioWg.Wait()

	select {
	case <-ctx.Done():
		return nil

	case out, ok := <-handle.DoneCh:
		if !ok {
			log.Println("done ch closed")
			return nil
		}
		log.Printf("done: %+v", out)
	}

	return nil
}

func redirect(ctx context.Context, wg *sync.WaitGroup, pipe *io.PipeReader, callback func([]byte) error) {
	defer wg.Done()

	buf := make([]byte, 1024)
	for {
		n, err := pipe.Read(buf)
		if err != nil {
			switch {
			case errors.Is(err, context.Canceled):
				// ignore
			case errors.Is(err, io.EOF):
				// ignore
			default:
				log.Printf("pipe read err: %+v", err)
			}
			return
		}

		log.Printf("sending: %d", n)
		if err := callback(buf[:n]); err != nil {
			return
		}
	}
}

type Language struct {
	ID         string
	ShowName   string
	Processors []Processor
}

type Processor struct {
	ID       string
	ShowName string

	DockerImage string

	DefaultFilename string

	Tasks []Task
}

type Task struct {
	ID       string
	ShowName string

	Kind string // "action" | "tool"

	Compile *PhasedTask
	Run     *PhasedTask
}

type PhasedTask struct {
	Cmd []string
}

var L = []Language{
	{
		ID:       "test-shell",
		ShowName: "Test Shell",

		Processors: []Processor{
			{
				ID:       "alpine-sh-latest",
				ShowName: "sh (alpine:latest)",

				DockerImage: "alpine:latest",

				DefaultFilename: "main.sh",

				Tasks: []Task{
					{
						ID:       "run",
						Kind:     "action",
						ShowName: "Run",

						Compile: nil,
						Run: &PhasedTask{
							Cmd: []string{"sh", "main.sh"},
						},
					},
				},
			},
		},
	},
}

func lookupLanguage(languageID, processorID, taskID string) (*Language, *Processor, *Task, error) {
	var lang *Language
	for _, l := range L {
		if l.ID == languageID {
			lang = &l
			break
		}
	}
	if lang == nil {
		return nil, nil, nil, errors.Errorf("language not found: '%s'", languageID)
	}

	var proc *Processor
	for _, p := range lang.Processors {
		if p.ID == processorID {
			proc = &p
			break
		}
	}
	if proc == nil {
		return nil, nil, nil, errors.Errorf("processor not found: '%s'", processorID)
	}

	var task *Task
	for _, t := range proc.Tasks {
		if t.ID == taskID {
			task = &t
			break
		}
	}
	if task == nil {
		return nil, nil, nil, errors.Errorf("task not found: '%s'", taskID)
	}

	return lang, proc, task, nil
}
