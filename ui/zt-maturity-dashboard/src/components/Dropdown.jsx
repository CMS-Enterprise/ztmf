import React, { Component } from "react";
// import Select from "react-select";
import { FixedSizeList as List } from "react-window";
import "../App.css";

const height = 35;

export default class MenuList extends Component {

  render() {
    const { options, children, maxHeight, getValue } = this.props;
    const [value] = getValue();
    const initialOffset = options.indexOf(value) * height;

    return (
      <List
        height={maxHeight}
        itemCount={children.length}
        itemSize={height}
        initialScrollOffset={initialOffset}
      >
        {({ index, style }) => <div style={style}>{children[index]}</div>}
      </List>
    );
  }
}

