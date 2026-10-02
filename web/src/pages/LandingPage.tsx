import { useState, type FormEvent, type ReactNode } from 'react'
import { Link, Navigate, useNavigate } from 'react-router'
import { Logo } from '../components/Logo'
import { SiteHeader } from '../components/SiteHeader'
import { StatusBadge } from '../components/StatusBadge'
import { homeFor, useAuth } from '../lib/auth'
import { isValidCode, normalizeCode } from '../lib/tracking'

const primary =
  'inline-flex min-h-11 items-center justify-center rounded-lg bg-brand-700 px-5 font-semibold text-white hover:bg-brand-800 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-600'
const secondary =
  'inline-flex min-h-11 items-center justify-center rounded-lg border border-slate-300 bg-white px-5 font-semibold text-slate-900 hover:bg-slate-50 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-600'

/** Página inicial: explica o produto e leva cada público à sua área. */
export function LandingPage() {
  const { state } = useAuth()
  // Quem já está logado vai direto para a própria área.
  if (state.status === 'authenticated') return <Navigate to={homeFor(state.session.role)} replace />

  return (
    <div className="flex min-h-dvh flex-col">
      <SiteHeader wide />
      <main className="flex-1">
        <Hero />
        <Audiences />
        <HowItWorks />
        <Features />
        <CallToAction />
      </main>
      <footer className="border-t border-slate-200 bg-white">
        <div className="mx-auto flex max-w-6xl flex-wrap items-center justify-between gap-3 px-4 py-6 text-sm text-slate-600">
          <span className="flex items-center gap-2 font-semibold text-brand-800">
            <Logo className="size-6" /> Rastreia
          </span>
          <span>Rastreio de entregas para transportadoras pequenas e médias.</span>
        </div>
      </footer>
    </div>
  )
}

function Hero() {
  return (
    <section className="bg-brand-800 text-white">
      <div className="mx-auto grid max-w-6xl items-center gap-10 px-4 py-12 md:grid-cols-[1.2fr_1fr] md:py-20">
        <div>
          <p className="font-semibold text-brand-50/80">Para transportadoras</p>
          <h1 className="mt-2 text-4xl leading-tight font-bold md:text-5xl">
            Seu cliente sabe onde está a encomenda. Sem ligar para ninguém.
          </h1>
          <p className="mt-4 max-w-xl text-lg text-brand-50/90">
            Cadastre as entregas, distribua para os motoristas e deixe o Rastreia avisar o cliente a cada passo, por
            e-mail e no celular, em tempo real.
          </p>
          <div className="mt-8 flex flex-wrap gap-3">
            <Link
              to="/transportadora/cadastro"
              className="inline-flex min-h-11 items-center justify-center rounded-lg bg-white px-5 font-semibold text-brand-800 hover:bg-brand-50 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white"
            >
              Cadastrar minha transportadora
            </Link>
            <Link
              to="/transportadora/entrar"
              className="inline-flex min-h-11 items-center justify-center rounded-lg border border-white/40 px-5 font-semibold hover:bg-white/10 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white"
            >
              Já tenho conta
            </Link>
          </div>
        </div>
        <TrackingPreview />
      </div>
    </section>
  )
}

/** Uma amostra da página que o cliente final vê, para mostrar o produto sem screenshot. */
function TrackingPreview() {
  const steps: { label: string; time: string; done: boolean }[] = [
    { label: 'Aguardando coleta', time: 'Hoje, 08:12', done: true },
    { label: 'Coletado', time: 'Hoje, 09:40', done: true },
    { label: 'Em rota', time: 'Hoje, 13:05', done: true },
    { label: 'Entregue', time: 'Previsto para hoje', done: false },
  ]
  return (
    <div aria-hidden="true" className="rounded-2xl bg-white p-5 text-slate-900 shadow-xl md:rotate-1">
      <p className="text-sm text-slate-600">Código RS7K2M9QXA4P</p>
      <p className="mt-1 text-xl font-bold">Olá, Maria</p>
      <p className="text-sm text-slate-600">Entrega feita por Expresso Sul</p>
      <div className="mt-3">
        <StatusBadge status="in_transit" size="lg" />
      </div>
      <ol className="mt-4 flex flex-col gap-3">
        {steps.map((s) => (
          <li key={s.label} className="flex items-center gap-3">
            <span className={`size-3 shrink-0 rounded-full ${s.done ? 'bg-brand-700' : 'border-2 border-slate-300'}`} />
            <span className={`flex-1 ${s.done ? 'font-medium' : 'text-slate-500'}`}>{s.label}</span>
            <span className="text-sm text-slate-500">{s.time}</span>
          </li>
        ))}
      </ol>
    </div>
  )
}

function Audiences() {
  return (
    <section className="mx-auto max-w-6xl px-4 py-12" aria-labelledby="audiences">
      <h2 id="audiences" className="text-2xl font-bold">
        Por onde você quer começar?
      </h2>
      <div className="mt-6 grid gap-4 md:grid-cols-3">
        <Card
          title="Sou transportadora"
          icon={<TruckIcon />}
          text="Cadastre motoristas e entregas, acompanhe tudo num painel e veja na hora o que atrasou ou falhou."
        >
          <Link to="/transportadora/cadastro" className={primary}>
            Criar conta
          </Link>
          <Link to="/transportadora/entrar" className={secondary}>
            Entrar
          </Link>
        </Card>
        <Card
          title="Sou motorista"
          icon={<PhoneIcon />}
          text="Abra no celular, veja as entregas do dia e mude o status com um toque, mesmo com sinal fraco."
        >
          <Link to="/motorista/entrar" className={primary}>
            Entrar no app
          </Link>
        </Card>
        <Card
          title="Quero rastrear"
          icon={<BoxIcon />}
          text="Recebeu um código por e-mail? Digite aqui para ver onde está sua encomenda, sem criar conta."
        >
          <TrackForm />
        </Card>
      </div>
    </section>
  )
}

function Card({ title, icon, text, children }: { title: string; icon: ReactNode; text: string; children: ReactNode }) {
  return (
    <article className="flex flex-col rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
      <span className="flex size-11 items-center justify-center rounded-lg bg-brand-50 text-brand-700">{icon}</span>
      <h3 className="mt-4 text-lg font-bold">{title}</h3>
      <p className="mt-1 flex-1 text-slate-600">{text}</p>
      <div className="mt-5 flex flex-wrap gap-2">{children}</div>
    </article>
  )
}

function TrackForm() {
  const navigate = useNavigate()
  const [code, setCode] = useState('')
  const [error, setError] = useState('')

  function submit(e: FormEvent) {
    e.preventDefault()
    const normalized = normalizeCode(code)
    if (!isValidCode(normalized)) {
      setError('O código tem 12 caracteres e começa com RS.')
      return
    }
    navigate(`/rastreio/${normalized}`)
  }

  return (
    <form onSubmit={submit} noValidate className="flex w-full flex-col gap-2">
      <label htmlFor="landing-code" className="sr-only">
        Código de rastreio
      </label>
      <div className="flex gap-2">
        <input
          id="landing-code"
          value={code}
          onChange={(e) => {
            setCode(e.target.value)
            setError('')
          }}
          placeholder="RS7K2M9QXA4P"
          autoComplete="off"
          autoCapitalize="characters"
          spellCheck={false}
          maxLength={20}
          aria-invalid={error ? true : undefined}
          aria-describedby={error ? 'landing-code-error' : undefined}
          className="min-h-11 w-full min-w-0 rounded-lg border border-slate-300 bg-white px-3 font-mono uppercase focus:border-brand-600 focus:outline-2 focus:outline-brand-600 aria-invalid:border-danger-fg"
        />
        <button type="submit" className={primary}>
          Rastrear
        </button>
      </div>
      {error && (
        <p id="landing-code-error" className="text-sm text-danger-fg">
          {error}
        </p>
      )}
    </form>
  )
}

function HowItWorks() {
  const steps = [
    { title: 'Cadastre', text: 'A transportadora cria a conta, cadastra os motoristas e lança as entregas.' },
    { title: 'Distribua', text: 'Cada motorista vê só as próprias entregas no celular, com endereço e mapa.' },
    { title: 'Acompanhe', text: 'Cada mudança de status chega ao painel e ao cliente na hora, com histórico.' },
  ]
  return (
    <section className="border-y border-slate-200 bg-white" aria-labelledby="how">
      <div className="mx-auto max-w-6xl px-4 py-12">
        <h2 id="how" className="text-2xl font-bold">
          Como funciona
        </h2>
        <ol className="mt-6 grid gap-6 md:grid-cols-3">
          {steps.map((s, i) => (
            <li key={s.title} className="flex gap-4">
              <span className="flex size-10 shrink-0 items-center justify-center rounded-full bg-brand-700 font-bold text-white">
                {i + 1}
              </span>
              <div>
                <h3 className="font-bold">{s.title}</h3>
                <p className="mt-1 text-slate-600">{s.text}</p>
              </div>
            </li>
          ))}
        </ol>
      </div>
    </section>
  )
}

function Features() {
  const items = [
    ['Tempo real', 'Painel e página de rastreio se atualizam sozinhos quando o motorista muda o status.'],
    ['Avisos ao cliente', 'E-mail a cada mudança e notificação no celular para quem ativar.'],
    ['Seus dados, só seus', 'Cada transportadora enxerga apenas os próprios motoristas e entregas.'],
    ['Privacidade', 'O link público não mostra endereço nem e-mail, expira depois da entrega e os dados são apagados no prazo.'],
  ]
  return (
    <section className="mx-auto max-w-6xl px-4 py-12" aria-labelledby="features">
      <h2 id="features" className="text-2xl font-bold">
        Feito para o dia a dia da entrega
      </h2>
      <dl className="mt-6 grid gap-4 sm:grid-cols-2">
        {items.map(([title, text]) => (
          <div key={title} className="rounded-xl border border-slate-200 bg-white p-5">
            <dt className="font-bold">{title}</dt>
            <dd className="mt-1 text-slate-600">{text}</dd>
          </div>
        ))}
      </dl>
    </section>
  )
}

function CallToAction() {
  return (
    <section className="mx-auto max-w-6xl px-4 pb-14">
      <div className="flex flex-col items-start gap-4 rounded-2xl bg-brand-50 p-6 md:flex-row md:items-center md:justify-between md:p-8">
        <div>
          <h2 className="text-2xl font-bold text-brand-800">Pronto para parar de responder "cadê minha encomenda?"</h2>
          <p className="mt-1 text-slate-600">Crie a conta da sua transportadora e lance a primeira entrega agora.</p>
        </div>
        <Link to="/transportadora/cadastro" className={primary}>
          Começar agora
        </Link>
      </div>
    </section>
  )
}

const iconProps = {
  viewBox: '0 0 24 24',
  className: 'size-6',
  fill: 'none',
  stroke: 'currentColor',
  strokeWidth: 2,
  strokeLinecap: 'round' as const,
  strokeLinejoin: 'round' as const,
  'aria-hidden': true,
}

function TruckIcon() {
  return (
    <svg {...iconProps}>
      <path d="M3 6h11v10H3zM14 10h4l3 3v3h-7" />
      <circle cx="7" cy="18" r="2" />
      <circle cx="17" cy="18" r="2" />
    </svg>
  )
}

function PhoneIcon() {
  return (
    <svg {...iconProps}>
      <rect x="7" y="2" width="10" height="20" rx="2" />
      <path d="M11 18h2" />
    </svg>
  )
}

function BoxIcon() {
  return (
    <svg {...iconProps}>
      <path d="m12 3 8 4.5v9L12 21l-8-4.5v-9z" />
      <path d="m4 7.5 8 4.5 8-4.5M12 12v9" />
    </svg>
  )
}
