// GameProvider — gắn state/nghiệp vụ từ các hook vào context.
import { useMemo } from 'react';

// Context
import { GameContext } from '../context';

// Hooks
import { useGame } from '../hooks/useGame';
import { useNotifications } from '../hooks/useNotifications';

const GameProvider = ({ children }) => {
  const game = useGame();
  useNotifications(game.scene); // đẩy toast server-emitted lên NotificationProvider (toàn cục)

  return (
    <GameContext.Provider value={useMemo(() => ({ ...game }), [game])}>
      {children}
    </GameContext.Provider>
  );
};

export default GameProvider;
