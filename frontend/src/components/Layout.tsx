import { Container, Navbar, Dropdown } from "react-bootstrap";
import { Outlet } from "react-router-dom";
import { Gear, BoxArrowRight } from "react-bootstrap-icons";
import { useContext } from "react";
import { UserInfoContext } from "./UserInfoProvider";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import ThemeToggle from "./ThemeToggle";

const Layout = () => {
  const { logout, userInfo, isAuthenticated } = useContext(UserInfoContext);
  const navigate = useNavigate();
  const { t } = useTranslation();

  const handleLogout = () => {
    logout();
    navigate("/login", { replace: true });
  };

  return (
    <>
      <Navbar
        bg="body-tertiary"
        className="shadow-sm"
      >
        <Container className="justify-content-end">
          <Navbar.Brand onClick={() => navigate("/dashboard")} role="button" className="p-0 me-auto">
            <img
              src="markpost.svg"
              alt="Markpost"
              height="40"
              className="d-inline-block"
              style={{ margin: 0 }}
            />
          </Navbar.Brand>
          <ThemeToggle />
          {isAuthenticated && (
            <Dropdown align="end">
              <Dropdown.Toggle
                variant="link"
                id="dropdown-basic"
                className="text-decoration-none d-flex align-items-center gap-2 ms-2 text-body"
              >
                <span className="d-none d-md-inline">
                  {userInfo?.user?.username || t("common.user")}
                </span>
              </Dropdown.Toggle>

              <Dropdown.Menu className="border-0 shadow-lg">
                <Dropdown.Item onClick={() => navigate("/settings")}>
                  <Gear size={16} className="me-2" />
                  {t("navigation.userMenu.settings")}
                </Dropdown.Item>
                <Dropdown.Divider />
                <Dropdown.Item className="text-danger" onClick={handleLogout}>
                  <BoxArrowRight size={16} className="me-2" />
                  {t("navigation.userMenu.logout")}
                </Dropdown.Item>
              </Dropdown.Menu>
            </Dropdown>
          )}
        </Container>
      </Navbar>
      <main>
        <Container>
          <Outlet />
        </Container>
      </main>
    </>
  );
};

export default Layout;
