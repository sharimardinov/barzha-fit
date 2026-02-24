/**
 * Animated Stepper — inspired by reactbits.dev/components/stepper
 * Adapted for BarzhaFit onboarding with accent color and no external CSS deps.
 */
import { useState, Children, useRef, useLayoutEffect } from "react";
import { motion as Motion, AnimatePresence } from "motion/react";

const stepVariants = {
  enter: (dir) => ({ x: dir >= 0 ? 80 : -80, opacity: 0 }),
  center: { x: 0, opacity: 1 },
  exit: (dir) => ({ x: dir >= 0 ? -80 : 80, opacity: 0 }),
};

function SlideTransition({ children, direction, onHeightReady }) {
  const ref = useRef(null);
  useLayoutEffect(() => {
    if (ref.current) onHeightReady(ref.current.offsetHeight);
  }, [children, onHeightReady]);

  return (
    <Motion.div
      ref={ref}
      custom={direction}
      variants={stepVariants}
      initial="enter"
      animate="center"
      exit="exit"
      transition={{ duration: 0.3, ease: [0.25, 0.1, 0.25, 1] }}
    >
      {children}
    </Motion.div>
  );
}

function StepContent({ isCompleted, currentStep, direction, children }) {
  const [height, setHeight] = useState("auto");

  return (
    <Motion.div
      style={{ position: "relative", overflow: "visible", minHeight: 0 }}
      animate={{ height: height || "auto" }}
      transition={{ duration: 0.3, ease: [0.25, 0.1, 0.25, 1] }}
    >
      <AnimatePresence initial={false} custom={direction} mode="wait">
        {!isCompleted && (
          <SlideTransition key={currentStep} direction={direction} onHeightReady={(h) => setHeight(h)}>
            {children}
          </SlideTransition>
        )}
      </AnimatePresence>
    </Motion.div>
  );
}

function StepIndicator({ step, currentStep, onClick }) {
  const status = currentStep === step ? "active" : currentStep > step ? "complete" : "inactive";

  return (
    <Motion.button
      onClick={() => onClick(step)}
      style={{
        position: "relative", border: "none", background: "transparent",
        cursor: "pointer", outline: "none", padding: 0,
      }}
      whileTap={{ scale: 0.9 }}
    >
      <Motion.div
        style={{
          width: 28, height: 28, borderRadius: "50%", display: "flex",
          alignItems: "center", justifyContent: "center", fontWeight: 600, fontSize: 12,
        }}
        animate={{
          backgroundColor: status === "active" ? "#ff033e" : status === "complete" ? "#ff033e" : "rgba(0,0,0,0.08)",
          color: status === "inactive" ? "rgba(0,0,0,0.4)" : "#fff",
          scale: status === "active" ? 1.1 : 1,
        }}
        transition={{ duration: 0.3 }}
      >
        {status === "complete" ? (
          <svg width={14} height={14} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={3} strokeLinecap="round" strokeLinejoin="round">
            <polyline points="20 6 9 17 4 12" />
          </svg>
        ) : status === "active" ? (
          <Motion.div
            style={{ width: 8, height: 8, borderRadius: "50%", background: "#fff" }}
            layoutId="active-dot"
          />
        ) : (
          <span>{step}</span>
        )}
      </Motion.div>
    </Motion.button>
  );
}

function StepConnector({ isComplete }) {
  return (
    <div style={{ flex: 1, height: 2, margin: "0 6px", borderRadius: 2, background: "rgba(0,0,0,0.08)", position: "relative", overflow: "hidden" }}>
      <Motion.div
        style={{ position: "absolute", left: 0, top: 0, height: "100%" }}
        animate={{ width: isComplete ? "100%" : 0, backgroundColor: isComplete ? "#ff033e" : "transparent" }}
        transition={{ duration: 0.4, ease: [0.4, 0, 0.2, 1] }}
      />
    </div>
  );
}

export function Step({ children }) {
  return <div>{children}</div>;
}

export default function Stepper({
  children,
  currentStep,
  onStepChange,
  onComplete,
  backText = "Назад",
  nextText = "Далее",
  completeText = "Готово",
  disableNav = false,
  loading = false,
  canProceed = true,
}) {
  const [direction, setDirection] = useState(0);
  const stepsArray = Children.toArray(children);
  const totalSteps = stepsArray.length;
  const isCompleted = currentStep > totalSteps;
  const isLastStep = currentStep === totalSteps;

  const goTo = (step) => {
    if (step === currentStep || loading) return;
    setDirection(step > currentStep ? 1 : -1);
    onStepChange(step);
  };

  const handleBack = () => {
    if (currentStep > 1) goTo(currentStep - 1);
  };

  const handleNext = () => {
    if (!canProceed || loading) return;
    if (isLastStep) {
      setDirection(1);
      onComplete();
    } else {
      goTo(currentStep + 1);
    }
  };

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 0 }}>
      {/* Step indicators */}
      <div style={{ display: "flex", alignItems: "center", padding: "0 0 20px", width: "100%" }}>
        {stepsArray.map((_, i) => {
          const stepNum = i + 1;
          return (
            <div key={i} style={{ display: "contents" }}>
              <StepIndicator step={stepNum} currentStep={currentStep} onClick={goTo} />
              {i < totalSteps - 1 && <StepConnector isComplete={currentStep > stepNum} />}
            </div>
          );
        })}
      </div>

      {/* Content */}
      <StepContent isCompleted={isCompleted} currentStep={currentStep} direction={direction}>
        {stepsArray[currentStep - 1]}
      </StepContent>

      {/* Footer */}
      {!isCompleted && !disableNav && (
        <div style={{ display: "flex", justifyContent: currentStep > 1 ? "space-between" : "flex-end", alignItems: "center", marginTop: 24 }}>
          {currentStep > 1 && (
            <button
              onClick={handleBack}
              style={{
                background: "transparent", border: "none", cursor: "pointer",
                color: "rgba(0,0,0,0.4)", fontWeight: 500, fontSize: 14, padding: "6px 12px",
                transition: "color 0.2s",
              }}
            >{backText}</button>
          )}
          <button
            onClick={handleNext}
            disabled={!canProceed || loading}
            style={{
              background: canProceed ? "#ff033e" : "rgba(0,0,0,0.1)",
              color: canProceed ? "#fff" : "rgba(0,0,0,0.3)",
              border: "none", borderRadius: 999, padding: "10px 24px",
              fontWeight: 600, fontSize: 14, cursor: canProceed ? "pointer" : "default",
              transition: "all 0.2s", letterSpacing: "-0.01em",
              boxShadow: canProceed ? "0 4px 12px rgba(255,3,62,0.25)" : "none",
              opacity: loading ? 0.7 : 1,
            }}
          >{isLastStep ? (loading ? "Сохраняю..." : completeText) : nextText}</button>
        </div>
      )}
    </div>
  );
}
