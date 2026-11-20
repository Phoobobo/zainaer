export default defineNuxtRouteMiddleware(async to => {
  const ctx = useAuthContext();
  const redirectTo = useState<string | null>("authRedirect", () => null);

  function redirectToLogin() {
    const token = to.query.token;
    if (typeof token === "string" && token && ["/join", "/items", "/home"].includes(to.path.replace(/\/$/, ""))) {
      // The registration form passes this token to the server. Saving /join
      // as the post-login destination would offer the same invitation twice.
      redirectTo.value = null;
      return navigateTo({ path: "/", query: { token } });
    }

    redirectTo.value = to.fullPath;
    return navigateTo("/");
  }

  if (!ctx.isAuthorized()) return redirectToLogin();

  if (!ctx.user) {
    const { data, error } = await useUserApi().user.self();
    if (error) return redirectToLogin();
    ctx.user = data.item;
  }
});
