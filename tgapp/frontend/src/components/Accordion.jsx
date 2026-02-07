import { useState, useRef, useCallback } from "react";

export function AccordionItem({ title, children, defaultOpen = false }) {
  const [open, setOpen] = useState(defaultOpen);
  const bodyRef = useRef(null);

  const toggle = useCallback(() => {
    setOpen((prev) => !prev);
  }, []);

  return (
    <div className={`accordion-item${open ? " open" : ""}`}>
      <button className="accordion-toggle" onClick={toggle}>
        <span>{title}</span>
        <div className="burger">
          <span /><span /><span />
        </div>
      </button>
      <div
        className="accordion-body"
        ref={bodyRef}
        style={{ maxHeight: open ? bodyRef.current?.scrollHeight + "px" : "0px" }}
      >
        {children}
      </div>
    </div>
  );
}

export function Accordion({ children }) {
  return <div className="accordion">{children}</div>;
}
