export default function Footer() {
  return (
    <footer className="border-t border-border bg-card">
      <div className="container py-6">
        <p className="text-center text-xs text-muted-foreground">
          © {new Date().getFullYear()} SkillZone. All rights reserved.
        </p>
      </div>
    </footer>
  );
}
