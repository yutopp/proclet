import React from 'react'
import { config } from "./constants";
import RunButton from './RunButton'
import { Language, Processor, Task } from "./proto/api/v1/server_pb";
import { AppBar, FormControl, InputLabel, MenuItem, Select, Toolbar, Typography } from "@mui/material";

export type Props = {
  languages: Language[];
  executionState: boolean;
  handleButtonClick: (langId: string, procId: string, taskId: string) => void;
};

export const Header = ({ languages, executionState, handleButtonClick }: Props) => {
  const [selectedLanguageId, setSelectedLanguageId] = React.useState<string>("");
  const [selectedProcessorId, setSelectedProcessorId] = React.useState<string>("");
  const [selectedTaskId, setSelectedTaskId] = React.useState<string>("");

  let processors: Processor[] = [];
  {
    let lang = languages.find((lang) => lang.id == selectedLanguageId);
    if (lang == null && languages.length > 0) {
      lang = languages[0];
      setSelectedLanguageId(lang.id);
    }

    if (lang != null) {
      processors = lang.processors;
    }
  }

  let tasks: Task[] = [];
  {
    let proc = processors.find((proc) => proc.id == selectedProcessorId);
    if (proc == null && processors.length > 0) {
      proc = processors[0];
      setSelectedProcessorId(proc.id);
    }

    if (proc != null) {
      tasks = proc.tasks;
    }
  }

  {
    let task = tasks.find((task) => task.id == selectedTaskId);
    if (task == null && tasks.length > 0) {
      task = tasks[0];
      setSelectedTaskId(task.id);
    }
  }

  let runAction = () => {
    handleButtonClick(selectedLanguageId, selectedProcessorId, selectedTaskId);
  };

  return (
    <>
      <AppBar position="sticky">
        <Toolbar>
          <Typography
            variant="h5"
            noWrap
            component="a"
            href="/"
            sx={{ flexGrow: 1, fontWeight: 700, textDecoration: 'none', color: 'inherit' }}
          >
            Proclet (α) {config.PROD ? "" : "(dev)"}
          </Typography>

          <FormControl sx={{ m: 1, minWidth: 120 }} size="small" >
            <InputLabel id="select-language-label" className="labelInsideAppBar">Language</InputLabel>
            <Select
              labelId="select-language-label"
              id="select-language"
              value={selectedLanguageId}
              label="Language"
              className="selectInsideAppBar inputInsideAppBar iconInsideAppBar"
              onChange={(e) => setSelectedLanguageId(e.target.value)}
            >
              {
                languages.map((lang) => {
                  return <MenuItem key={lang.id} value={lang.id}>{lang.showName}</MenuItem>;
                })
              }
            </Select>
          </FormControl>

          <FormControl sx={{ m: 1, minWidth: 120 }} size="small">
            <InputLabel id="select-processor-label" className="labelInsideAppBar">Processor</InputLabel>
            <Select
              labelId="select-processor-label"
              id="select-processor"
              value={selectedProcessorId}
              label="Processor"
              className="selectInsideAppBar inputInsideAppBar iconInsideAppBar"
              onChange={(e) => setSelectedProcessorId(e.target.value)}
            >
              {
                processors.map((proc) => {
                  return <MenuItem key={`${selectedLanguageId}-${proc.id}`} value={proc.id}>{proc.showName}</MenuItem>;
                })
              }
            </Select>
          </FormControl>

          <RunButton
            defaultIndex={0}
            tasks={tasks}
            isExecuting={executionState}
            onChange={(id) => setSelectedTaskId(id)}
            onClick={runAction}
          ></RunButton>
        </Toolbar>
      </AppBar>
    </>
  );
};
