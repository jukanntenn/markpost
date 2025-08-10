import { Container, Navbar, Dropdown } from "react-bootstrap";
import { Outlet } from "react-router-dom";
import { Gear, BoxArrowRight } from "react-bootstrap-icons";
import { useContext } from "react";
import { UserInfoContext } from "./UserInfoProvider";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import ThemeToggle from "./ThemeToggle";
import LanguageToggle from "./LanguageToggle";

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
        bg="primary"
        data-bs-theme="dark"
        expand="lg"
        className="shadow-sm"
      >
        <Container>
          <Navbar.Brand href="#home" className="p-0">
            <img
              src="markpost.svg"
              alt="Markpost"
              height="40"
              className="d-inline-block"
              style={{ margin: 0 }}
            />
          </Navbar.Brand>
          <Navbar.Toggle
            aria-controls="basic-navbar-nav"
            className="border-0"
          />
          <Navbar.Collapse className="justify-content-end d-flex align-items-center">
            <ThemeToggle />
            <LanguageToggle />
            {isAuthenticated && (
              <Dropdown align="end">
                <Dropdown.Toggle
                  variant="link"
                  id="dropdown-basic"
                  className="text-decoration-none d-flex align-items-center gap-2 ms-2"
                >
                  <div
                    className="bg-primary rounded-circle d-flex align-items-center justify-content-center"
                    style={{ width: "32px", height: "32px" }}
                  >
                    <span className="fw-bold text-white">
                      {userInfo?.user?.username?.charAt(0).toUpperCase() || "U"}
                    </span>
                  </div>
                  <span className="d-none d-md-inline">
                    {userInfo?.user?.username || "User"}
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
          </Navbar.Collapse>
        </Container>
      </Navbar>
      <main className="min-vh-100">
        <Container fluid className="py-4">
          <Outlet />
        </Container>
      </main>
    </>
  );
};

export default Layout;
