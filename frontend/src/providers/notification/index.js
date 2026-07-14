// NotificationProvider — sở hữu hàng đợi toast toàn cục + render NotificationStack.
// Bọc NGOÀI CÙNG để mọi lớp dưới (Authentication, Game…) gọi được notify().
import { useCallback, useMemo, useRef, useState } from 'react';

// Contexts
import { NotificationContext } from '@contexts/notification';

// Components
import NotificationStack from '@components/notificationStack';

const NotificationProvider = ({ children }) => {
  const [items, setItems] = useState([]);
  const seq = useRef(0);

  const dismiss = useCallback((id) => {
    setItems((prev) => prev.filter((n) => n.id !== id));
  }, []);

  const notify = useCallback((n) => {
    setItems((prev) => [...prev, { ...n, id: ++seq.current }]);
  }, []);

  return (
    <NotificationContext.Provider
      value={useMemo(() => ({ items, notify, dismiss }), [items, notify, dismiss])}
    >
      {children}
      <NotificationStack items={items} onDismiss={dismiss} />
    </NotificationContext.Provider>
  );
};

export default NotificationProvider;
