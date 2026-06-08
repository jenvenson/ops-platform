// Global type declarations for the project

// Declare module for JSX files
declare module '*.jsx' {
  import { ComponentType } from 'react';

  const component: ComponentType<any>;
  export default component;
}

// Declare module for specific AIChatbot component
declare module './AIChatbot' {
  import { ComponentType } from 'react';

  const AIChatbot: ComponentType<any>;
  export default AIChatbot;
}