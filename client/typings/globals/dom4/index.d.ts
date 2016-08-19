// Generated by typings
// Source: https://raw.githubusercontent.com/DefinitelyTyped/DefinitelyTyped/c00b5a1d9eeb8daae9dfccf502559929c2538e41/dom4/dom4.d.ts
interface ParentNode {
    /**
     * Returns the child elements.
     */
    children: HTMLCollection;

    /**
     * Returns the first element that is a descendant of node that matches relativeSelectors.
     */
    query(relativeSelectors: string): Element;

    /**
     * Returns all element descendants of node that match relativeSelectors.
     */
    queryAll(relativeSelectors: string): Elements;
}

interface Element extends ParentNode {
    /**
     * Returns the first (starting at element) inclusive ancestor that matches selectors, and null otherwise.
     */
    closest(selectors: string): Element;

    /**
     * Returns true if matching selectors against element’s root yields element, and false otherwise.
     */
    matches(selectors: string): boolean;
}

interface Elements extends ElementTraversal, ParentNode, Array<Element> {
}

interface Document extends ElementTraversal, ParentNode {
}

interface DocumentFragment extends ElementTraversal, ParentNode {
}
