// Context toàn cục cho toast in-app (chỉ định nghĩa). Triển khai ở @providers/notification.
import { createContext, useContext } from 'react';

const NotificationContext = createContext({});
const useNotificationContext = () => useContext(NotificationContext);

export { NotificationContext, useNotificationContext };
