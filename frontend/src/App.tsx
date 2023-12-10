import { useState } from 'react'
import { KoyaServiceClient } from "./proto/api/v1/server.client";
import { GrpcWebFetchTransport } from "@protobuf-ts/grpcweb-transport";
import './App.css'

const client = new KoyaServiceClient(
  new GrpcWebFetchTransport({
    baseUrl: 'http://localhost:9000'
  })
);

function App() {
  const [count, setCount] = useState(0);

  return (
    <>
      <h1>Torigoya</h1>
      <div className="card">
        <button onClick={() => {
          let stream = client.runOneshot({
            code: "print('hello world')",
          });
          stream.responses.onMessage((msg) => {
            console.log(msg);
          });
          setCount((count) => count + 1)
        }}>
          count is {count}
        </button>
      </div>
    </>
  )
}

export default App
