import { Link } from "react-router-dom";
import { Award, CheckCircle, Users, ArrowRight } from "lucide-react";
import heroBg from "../assets/hero-bg.jpg";

const steps = [
  {
    icon: Users,
    title: "Join Events",
    description:
      "Discover and participate in tech events, internships, and skill-building opportunities.",
  },
  {
    icon: CheckCircle,
    title: "Get Verified",
    description:
      "Organizations verify your participation and confirm your skills completion.",
  },
  {
    icon: Award,
    title: "Earn Badges",
    description:
      "Receive verified badges that showcase your skills to potential employers.",
  },
];

export default function Home() {
  return (
    <div>
      {/* Hero */}
      <section
        className="relative flex min-h-[85vh] items-center justify-center overflow-hidden bg-cover bg-center"
        style={{ backgroundImage: `url(${heroBg})` }}
      >
        <div className="absolute inset-0 hero-gradient opacity-85" />
        <div className="relative z-10 container text-center">
          <h1 className="animate-fade-in mb-6 text-4xl font-extrabold leading-tight text-primary-foreground md:text-6xl lg:text-7xl">
            Verify Your Skills.
            <br />
            Unlock Opportunities.
          </h1>
          <p className="animate-fade-in-delay mx-auto mb-10 max-w-2xl text-lg text-primary-foreground/80 md:text-xl">
            SkillZone connects individuals with events and organizations to
            build credible, verified skill profiles that employers trust.
          </p>
          <div className="animate-fade-in-delay flex flex-col items-center gap-4 sm:flex-row sm:justify-center">
            <Link
              to="/signup"
              className="inline-flex items-center gap-2 rounded-xl bg-background px-8 py-3.5 text-sm font-semibold text-primary shadow-lg transition hover:shadow-xl"
            >
              Get Started <ArrowRight className="h-4 w-4" />
            </Link>
            <Link
              to="/about"
              className="inline-flex items-center gap-2 rounded-xl border-2 border-primary-foreground/30 px-8 py-3.5 text-sm font-semibold text-primary-foreground transition hover:bg-primary-foreground/10"
            >
              Learn More
            </Link>
          </div>
        </div>
      </section>

      {/* What is SkillZone */}
      <section className="section-padding bg-background">
        <div className="container max-w-3xl text-center">
          <h2 className="mb-4 text-3xl font-bold text-foreground md:text-4xl">
            What is <span className="text-gradient">SkillZone</span>?
          </h2>
          <p className="text-lg leading-relaxed text-muted-foreground">
            SkillZone is a skill verification and opportunity platform that
            bridges the gap between talent and employers. Attend events, earn
            verified badges, and build a credible professional profile — all
            in one place.
          </p>
        </div>
      </section>

      {/* How It Works */}
      <section className="section-padding bg-card">
        <div className="container">
          <h2 className="mb-12 text-center text-3xl font-bold text-foreground md:text-4xl">
            How It Works
          </h2>
          <div className="grid gap-8 md:grid-cols-3">
            {steps.map((step, i) => (
              <div
                key={i}
                className="card-shadow rounded-2xl bg-background p-8 text-center transition hover:card-shadow-hover"
              >
                <div className="mx-auto mb-5 flex h-14 w-14 items-center justify-center rounded-xl bg-accent">
                  <step.icon className="h-7 w-7 text-accent-foreground" />
                </div>
                <h3 className="mb-2 text-xl font-semibold text-foreground">
                  {step.title}
                </h3>
                <p className="text-sm leading-relaxed text-muted-foreground">
                  {step.description}
                </p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* CTA */}
      <section className="section-padding hero-gradient">
        <div className="container text-center">
          <h2 className="mb-4 text-3xl font-bold text-primary-foreground md:text-4xl">
            Ready to Build Your Credibility?
          </h2>
          <p className="mx-auto mb-8 max-w-xl text-primary-foreground/80">
            Join SkillZone today and start earning verified skill badges that
            make your profile stand out to employers.
          </p>
          <Link
            to="/signup"
            className="inline-flex items-center gap-2 rounded-xl bg-background px-8 py-3.5 text-sm font-semibold text-primary shadow-lg transition hover:shadow-xl"
          >
            Create Your Account <ArrowRight className="h-4 w-4" />
          </Link>
        </div>
      </section>
    </div>
  );
}
