//import { useState } from 'react'
import { KoyaServiceClient } from "./proto/api/v1/server.client";
import { GrpcWebFetchTransport } from "@protobuf-ts/grpcweb-transport";
import React from 'react'
import { Terminal } from 'xterm'
import { config } from "./constants";
import './App.css'
import 'xterm/css/xterm.css';

type HeaderProps = {
  executionState: boolean;
  handleButtonClick: () => void;
};

const Header = ({ executionState, handleButtonClick }: HeaderProps) => {
  return (
    <header className="py-4 bg-blue-500 text-white text-center flex justify-between items-center">
      <h1 className="text-2xl font-bold ml-4">Proclet {config.PROD ? "" : "(dev)"}</h1>
      <button className="px-4 py-2 bg-white text-blue-500 rounded mr-4" onClick={handleButtonClick}>{executionState ? "!" : "?"}Run</button>
    </header>
  );
};

function App() {
  const ref = React.useRef<HTMLDivElement>(null);
  const termRef = React.useRef<{ term: Terminal, open: boolean }>({ term: new Terminal({convertEol: true}), open: false });
  const abortControllerRef = React.useRef<AbortController>(new AbortController());
  const [executionState, setExecutionState] = React.useState(false);
  const [mainSource, setMainSource] = React.useState("ulimit -a; uname -a; whoami; sleep 5; echo hello");

  React.useEffect(() => {
    if (termRef.current.open == false) {
      termRef.current.open = true;
      termRef.current.term.open(ref.current!);
    }
  }, [])

  const fetchData = async (source: string, signal: AbortSignal) => {
    const client = new KoyaServiceClient(
      new GrpcWebFetchTransport({
        baseUrl: config.BACKEND_URL,
      })
    );

    try {
      const call = client.runOneshot({
        code: source,
      }, {
        abort: signal,
      });
      for await (const message of call.responses) {
        console.log("got a message", message)
        switch (message.response.oneofKind) {
          case "output":
            // TODO: stderr
            termRef.current.term.write(message.response.output.buffer);
            break;
        }
      }
      let { status, trailers } = await call;
      console.log("status", status);
      console.log("trailers", trailers);
    }
    catch (e) {
      console.error(e);
    }
  };

  const handleButtonClick = () => {
    abortControllerRef.current.abort();
    termRef.current.term.reset();
    setExecutionState(true);

    const newAbortController = new AbortController();
    abortControllerRef.current = newAbortController;

    const source = mainSource;
    fetchData(source, newAbortController.signal).finally(() => {
      setExecutionState(false);
    });
  };

  return (
    <>
      <div className="min-h-screen flex flex-col bg-gray-100">
        <Header executionState={executionState} handleButtonClick={handleButtonClick} />

        {/* Main Content */}
        <div className="flex-1 flex bg-gray-100">
          {/* Left Section */}
          <div className="flex-1 flex flex-col items-center justify-center">
            <textarea
              className="w-full h-full p-4 border rounded"
              placeholder="Enter your text here..."
              defaultValue={mainSource}
              onChange={(e) => setMainSource(e.target.value)}
            />
          </div>

          {/* Right Section */}
          <div className="flex-1 flex flex-col items-center justify-center">
            <div
              ref={ref}
              className="w-full h-full p-4 border rounded"
            ></div>
          </div>
        </div>
      </div>
    </>
  )
}

export default App
