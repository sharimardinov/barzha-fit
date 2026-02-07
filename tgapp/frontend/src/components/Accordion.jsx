import { useState, useRef, useEffect, useCallback } from "react";

export function AccordionItem({ title, children, defaultOpen = false }) {
  const [open, setOpen] = useState(defaultOpen);
  const bodyRef = useRef(null);

  const toggle = useCallback(() => {
    const body = bodyRef.current;
    if (!body) { setOpen((p) => !p); return; }

    if (open) {
      // Close: set explicit height first, then collapse
      body.style.height = body.scrollHeight + "px";
      requestAnimationFrame(() => { body.style.height = "0px"; });
      setOpen(false);
    } else {
      // Open: expand to scrollHeight
      body.style.height = body.scrollHeight + "px";
      setOpen(true);
    }
  }, [open]);

  // After transition ends, set height to auto so content can resize
  useEffect(() => {
    const body = bodyRef.current;
    if (!body) return;
    const handler = (e) => {
      if (e.propertyName !== "height") return;
      if (open) body.style.height = "auto";
    };
    body.addEventListener("transitionend", handler);
    return () => body.removeEventListener("transitionend", handler);
  }, [open]);

  // Initialize: if defaultOpen, set auto height
  useEffect(() => {
    if (defaultOpen && bodyRef.current) {
      bodyRef.current.style.height = "auto";
    }
  }, [defaultOpen]);

  return (
    <div className={`accordion-item${open ? " open" : ""}`}>
      <button className="accordion-toggle" onClick={toggle} type="button">
        <span>{title}</span>
        <div className="burger">
          <span /><span /><span />
        </div>
      </button>
      <div className="accordion-body" ref={bodyRef} style={{ height: defaultOpen ? "auto" : "0px" }}>
        {children}
      </div>
    </div>
  );
}

export function Accordion({ children }) {
  return <div className="accordion">{children}</div>;
}
