
export default function Footer() {
    return (
        <footer className="border-t border-white/10 py-12 text-center">
            <p className="text-default-400 text-sm">
                Built with ❤️ by <a href="https://pramirez.dev" target="_blank" rel="noopener noreferrer" className="underline">Pablo Ramirez</a> | <a href="https://github.com/pablor21" target="_blank" rel="noopener noreferrer" className="underline">GitHub</a>
            </p>
            <p className="text-default-400 text-sm">
                Docs by <a href="https://github.com/Dan6erbond" target="_blank" rel="noopener noreferrer" className="underline">RaviAnand</a>.
            </p>
            <p className="text-default-400 text-sm">
                © {new Date().getFullYear()} GQLSchemaGen. Open Source MIT License.
            </p>
        </footer>
    );
}