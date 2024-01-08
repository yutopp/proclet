import React from 'react'
import { Code, ConnectError } from "@connectrpc/connect";
import { RunnerService } from "./proto/api/v1/server_connect";
import { Header } from './Header'
import './App.css'
import { Terminal } from 'xterm'
import 'xterm/css/xterm.css';
import { Language } from "./proto/api/v1/server_pb";
import { Box, Container, Grid } from "@mui/material";
import { useClient } from "./client";
import { EditorView } from '@codemirror/view'
import { EditorState } from '@codemirror/state'
import { basicSetup } from 'codemirror'

function encodeUTF8(str: string): Uint8Array {
  const encoder = new TextEncoder();
  return encoder.encode(str);
}

function App() {
  const ref = React.useRef<HTMLDivElement>(null);
  const termRef = React.useRef<{ term: Terminal, open: boolean }>({ term: new Terminal({ convertEol: true }), open: false });

  const editorDOMRef = React.useRef<HTMLDivElement>(null);
  const editorRef = React.useRef<{ view: EditorView | null, created: boolean }>({ view: null, created: false });

  const abortControllerRef = React.useRef<AbortController>(new AbortController());
  const [executionState, setExecutionState] = React.useState(false);
  const [mainSource, setMainSource] = React.useState("ulimit -a; uname -a; whoami; sleep 5; echo hello");
  const [languages, setLanguages] = React.useState<Language[]>([]);
  const client = useClient(RunnerService);

  React.useEffect(() => {
    const abort = new AbortController();
    const asyncFn = async () => {
      try {
        const resp = await client.list({}, { signal: abort.signal });
        setLanguages(resp.languages);
      } catch (e) {
        console.error(e);
      }
    };
    asyncFn();
  }, []);

  React.useEffect(() => {
    if (!termRef.current.open && ref.current != null) {
      termRef.current.open = true;
      termRef.current.term.open(ref.current!);
    }
  }, []);

  const runOneshot = async (langId: string, procId: string, taskId: string, source: string, signal: AbortSignal) => {
    try {
      let lang = languages.find((lang) => lang.id == langId);
      if (lang == null) {
        throw new Error("language not found");
      }
      let proc = lang.processors.find((proc) => proc.id == procId);
      if (proc == null) {
        throw new Error("processor not found");
      }
      let task = proc.tasks.find((task) => task.id == taskId);
      if (task == null) {
        throw new Error("task not found");
      }

      const opts = {
        signal: signal,
      };
      const call = client.runOneshot({
        languageId: langId,
        processorId: procId,
        taskId: taskId,

        files: [
          {
            path: proc.defaultFilename,
            content: encodeUTF8(source),
          },
        ],
      }, opts);
      for await (const message of call) {
        switch (message.response.case) {
          case "output":
            // TODO: stderr
            termRef.current.term.write(message.response.value.buffer);
            break;
        }
      }
    }
    catch (e) {
      if (e instanceof ConnectError) {
        switch (e.code) {
          case Code.Canceled:
            break; // ignore
          default:
            console.error("connect error", e.code, e.message, e.metadata);
            break;
        }
      } else {
        console.error("general error", e);
      }
    }
  };

  React.useEffect(() => {
    if (editorRef.current.created || editorDOMRef.current == null) {
      return;
    }

    console.log("creating editor");

    const updateCallback = EditorView.updateListener.of((update) => {
      if (update.docChanged) {
        setMainSource(update.state.doc.toString());
      }
    });

    const fixedHeightEditor = EditorView.theme({
      "&": { height: "100%", maxHeight: "100%" },
      ".cm-scroller": { overflow: "auto" }
    })

    const state = EditorState.create({
      doc: mainSource,
      extensions: [
        basicSetup,
        updateCallback,
        fixedHeightEditor,
      ],
    });

    const view = new EditorView({
      state,
      parent: editorDOMRef.current,
    });
    editorRef.current.created = true;
    editorRef.current.view = view;

    return () => {
      // view.destroy();
      // editor.current.removeEventListener("input", log);
    };
  }, []);

  const handleButtonClick = (langId: string, procId: string, taskId: string) => {
    abortControllerRef.current?.abort();

    termRef.current.term.reset();
    setExecutionState(true);

    const newAbortController = new AbortController();
    abortControllerRef.current = newAbortController;

    const source = mainSource;
    runOneshot(langId, procId, taskId, source, newAbortController.signal).finally(() => {
      setExecutionState(false);
    });
  };

  return (
    <Box display="flex" flexDirection="column" height="100vh">
      <Header languages={languages} executionState={executionState} handleButtonClick={handleButtonClick} />

      <Container component="main" maxWidth={false} sx={{ flexGrow: 1, overflow: 'auto', height: 'calc(100vh - 64px)' }}>
        {/* Grid to split the container horizontally */}
        <Grid container spacing={2} sx={{ height: 'calc(100vh - 64px)' }}>

          {/* Left half */}
          <Grid item xs={6}>
            <Box sx={{ height: 'calc(100vh - 64px)' }}>
              <div ref={editorDOMRef} className="w-full h-full"></div>
            </Box>
          </Grid>

          {/* Right half */}
          <Grid item xs={6}>
            <Box sx={{ height: 'calc(100vh - 64px)' }}>
              <div ref={ref} className="w-full h-full"></div>
            </Box>
          </Grid>
        </Grid>
      </Container>
    </Box>
  );
}

export default App
