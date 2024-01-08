import * as React from 'react';
import { Button, ButtonGroup, Grow, Paper, Popper, MenuItem, MenuList, CircularProgress } from '@mui/material';
import ArrowDropDownIcon from '@mui/icons-material/ArrowDropDown';
import ClickAwayListener from '@mui/material/ClickAwayListener';
import { Task } from './proto/api/v1/server_pb';
import { blue } from '@mui/material/colors';

type Props = {
  defaultIndex: number;
  tasks: Task[];
  isExecuting: boolean;
  onChange: (id: string) => void;
  onClick: () => void;
};

export default function RunButton({
  defaultIndex, tasks, isExecuting,
  onClick, onChange,
}: Props) {
  const [open, setOpen] = React.useState(false);
  const anchorRef = React.useRef<HTMLDivElement>(null);

  const [selectedIndex, setSelectedIndex] = React.useState(defaultIndex);

  const handleClick = () => onClick();

  // console.info(`You clicked ${options[selectedIndex]}`);

  const handleMenuItemClick = (
    _event: React.MouseEvent<HTMLLIElement, MouseEvent>,
    index: number,
  ) => {
    setSelectedIndex(index);
    setOpen(false);

    onChange(tasks[index].id);
  };

  const handleToggle = () => {
    setOpen((prevOpen) => !prevOpen);
  };

  const handleClose = (event: Event) => {
    if (
      anchorRef.current &&
      anchorRef.current.contains(event.target as HTMLElement)
    ) {
      return;
    }

    setOpen(false);
  };

  return (
    <React.Fragment>
      <ButtonGroup variant="contained" ref={anchorRef} aria-label="action button">
        <Button onClick={handleClick}>
          {tasks[selectedIndex]?.showName ?? ""}
          {isExecuting && (
            <CircularProgress
              size={24}
              sx={{
                color: blue[800],
                position: 'absolute',
                top: '50%',
                left: '50%',
                marginTop: '-12px',
                marginLeft: '-12px',
              }}
            />
          )}
        </Button>
        <Button
          size="small"
          aria-controls={open ? 'split-button-menu' : undefined}
          aria-expanded={open ? 'true' : undefined}
          aria-label="select actions"
          aria-haspopup="menu"
          onClick={handleToggle}
        >
          <ArrowDropDownIcon />
        </Button>
      </ButtonGroup>
      <Popper
        sx={{ zIndex: 1 }}
        open={open}
        anchorEl={anchorRef.current}
        role={undefined}
        transition
        disablePortal
      >
        {({ TransitionProps, placement }) => (
          <Grow
            {...TransitionProps}
            style={{
              transformOrigin:
                placement === 'bottom' ? 'center top' : 'center bottom',
            }}
          >
            <Paper>
              <ClickAwayListener onClickAway={handleClose}>
                <MenuList id="split-button-menu" autoFocusItem>
                  {tasks.map((task, index) => (
                    <MenuItem
                      key={task.id}
                      selected={index === selectedIndex}
                      onClick={(event) => handleMenuItemClick(event, index)}
                    >
                      {task.showName}
                    </MenuItem>
                  ))}
                </MenuList>
              </ClickAwayListener>
            </Paper>
          </Grow>
        )}
      </Popper>
    </React.Fragment>
  );
}
