// Context phiên đăng nhập (chỉ định nghĩa). Triển khai ở @providers/authentication.
import { createContext, useContext } from 'react';

const AuthenticationContext = createContext({});
const useAuthenticationContext = () => useContext(AuthenticationContext);

export { AuthenticationContext, useAuthenticationContext };
