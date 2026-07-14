// Context của feature "game" (chỉ createContext + hook tiêu thụ).
// Bản triển khai (state/logic) nằm ở ../provider + ../hooks.
import { createContext, useContext } from 'react';

const GameContext = createContext({});
const useGameContext = () => useContext(GameContext);

export { GameContext, useGameContext };
